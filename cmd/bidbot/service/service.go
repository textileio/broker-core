package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	st "github.com/textileio/broker-core/cmd/bidbot/service/store"
	"github.com/textileio/broker-core/dshelper/txndswrap"
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1/message"
	"github.com/textileio/broker-core/marketpeer"
	"google.golang.org/protobuf/proto"
)

var (
	log = golog.Logger("bidbot/service")

	// bidsAckTimeout is the max duration bidbot will wait for an ack after bidding in an auction.
	bidsAckTimeout = time.Second * 10
)

// Config defines params for Service configuration.
type Config struct {
	RepoPath       string
	Peer           marketpeer.Config
	BidParams      BidParams
	AuctionFilters AuctionFilters
}

// BidParams defines how bids are made.
type BidParams struct {
	MinerAddr     string
	WalletAddrSig []byte

	AskPrice         int64 // attoFIL per GiB per epoch
	VerifiedAskPrice int64 // attoFIL per GiB per epoch
	FastRetrieval    bool
	DealStartWindow  uint64 // number of epochs after which won deals must start be on-chain
}

// Validate ensures BidParams are valid.
func (p *BidParams) Validate() error {
	if p.DealStartWindow == 0 {
		return fmt.Errorf("invalid deal start window; must be greater than zero")
	}
	return nil
}

// AuctionFilters specifies filters used when selecting auctions to bid on.
type AuctionFilters struct {
	DealDuration MinMaxFilter
	DealSize     MinMaxFilter
}

// Validate ensures AuctionFilters are valid.
func (f *AuctionFilters) Validate() error {
	if err := f.DealDuration.Validate(); err != nil {
		return fmt.Errorf("invalid deal duration filter: %v", err)
	}
	if err := f.DealDuration.Validate(); err != nil {
		return fmt.Errorf("invalid deal size filter: %v", err)
	}
	return nil
}

// MinMaxFilter is used to specify a range for an auction filter.
type MinMaxFilter struct {
	Min uint64
	Max uint64
}

// Validate ensures the filter is a valid min max window.
func (f *MinMaxFilter) Validate() error {
	if f.Min > f.Max {
		return errors.New("min must be less than or equal to max")
	}
	return nil
}

// Service is a miner service that subscribes to brokered deals.
type Service struct {
	peer       *marketpeer.Peer
	fc         auctioneer.FilClient
	store      *st.Store
	subscribed bool

	bidParams      BidParams
	auctionFilters AuctionFilters

	ctx       context.Context
	finalizer *finalizer.Finalizer
	lk        sync.Mutex
}

// New returns a new Service.
func New(conf Config, store txndswrap.TxnDatastore, fc auctioneer.FilClient) (*Service, error) {
	if err := conf.BidParams.Validate(); err != nil {
		return nil, fmt.Errorf("validating bid parameters: %v", err)
	}
	if err := conf.AuctionFilters.Validate(); err != nil {
		return nil, fmt.Errorf("validating auction filters: %v", err)
	}

	fin := finalizer.NewFinalizer()
	ctx, cancel := context.WithCancel(context.Background())
	fin.Add(finalizer.NewContextCloser(cancel))

	// Create local datastore
	// s := st.NewStore(store)
	// fin.Add(s)

	// Create miner peer
	p, err := marketpeer.New(conf.Peer)
	if err != nil {
		return nil, fin.Cleanupf("creating peer: %v", err)
	}
	fin.Add(p)

	// Verify miner address
	ok, err := fc.VerifyBidder(
		conf.BidParams.WalletAddrSig,
		p.Host().ID(),
		conf.BidParams.MinerAddr)
	if err != nil {
		return nil, fin.Cleanupf("verifying miner address: %v", err)
	}
	if !ok {
		return nil, fin.Cleanup(fmt.Errorf("invalid miner address or signature"))
	}

	srv := &Service{
		// store:          s,
		peer:           p,
		fc:             fc,
		bidParams:      conf.BidParams,
		auctionFilters: conf.AuctionFilters,
		ctx:            ctx,
		finalizer:      fin,
	}

	log.Info("service started")

	return srv, nil
}

// Close the service.
func (s *Service) Close() error {
	log.Info("service was shutdown")
	return s.finalizer.Cleanup(nil)
}

// Subscribe to the deal auction feed.
// If bootstrap is true, the peer will dial the configured bootstrap addresses
// before joining the deal auction feed.
func (s *Service) Subscribe(bootstrap bool) error {
	s.lk.Lock()
	defer s.lk.Unlock()
	if s.subscribed {
		return nil
	}

	// Bootstrap against configured addresses
	if bootstrap {
		s.peer.Bootstrap()
	}

	// Subscribe to the global auctions topic
	auctions, err := s.peer.NewTopic(s.ctx, broker.AuctionTopic, true)
	if err != nil {
		return fmt.Errorf("creating auctions topic: %v", err)
	}
	auctions.SetEventHandler(s.eventHandler)
	auctions.SetMessageHandler(s.auctionsHandler)

	// Subscribe to our own wins topic
	wins, err := s.peer.NewTopic(s.ctx, broker.WinsTopic(s.peer.Host().ID()), true)
	if err != nil {
		if err := auctions.Close(); err != nil {
			log.Errorf("closing auctions topic: %v", err)
		}
		return fmt.Errorf("creating wins topic: %v", err)
	}
	wins.SetEventHandler(s.eventHandler)
	wins.SetMessageHandler(s.winsHandler)

	// Subscribe to our own proposals topic
	props, err := s.peer.NewTopic(s.ctx, broker.ProposalsTopic(s.peer.Host().ID()), true)
	if err != nil {
		if err := auctions.Close(); err != nil {
			log.Errorf("closing auctions topic: %v", err)
		}
		if err := wins.Close(); err != nil {
			log.Errorf("closing wins topic: %v", err)
		}
		return fmt.Errorf("creating proposals topic: %v", err)
	}
	props.SetEventHandler(s.eventHandler)
	props.SetMessageHandler(s.proposalHandler)

	s.finalizer.Add(auctions)
	s.finalizer.Add(wins)
	s.finalizer.Add(props)

	log.Info("subscribed to the deal auction feed")

	s.subscribed = true
	return nil
}

func (s *Service) eventHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s peer event: %s %s", topic, from, msg)
}

func (s *Service) auctionsHandler(from peer.ID, topic string, msg []byte) ([]byte, error) {
	log.Debugf("%s received auction from %s", topic, from)

	auction := &pb.Auction{}
	if err := proto.Unmarshal(msg, auction); err != nil {
		return nil, fmt.Errorf("unmarshaling message: %v", err)
	}

	auctionj, err := json.MarshalIndent(auction, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling json: %v", err)
	}
	log.Infof("received auction %s from %s: \n%s", auction.Id, from, string(auctionj))

	go func() {
		if err := s.makeBid(auction, from); err != nil {
			log.Errorf("making bid: %v", err)
		}
	}()
	return nil, nil
}

func (s *Service) winsHandler(from peer.ID, topic string, msg []byte) ([]byte, error) {
	log.Debugf("%s received win from %s", topic, from)

	win := &pb.WinningBid{}
	if err := proto.Unmarshal(msg, win); err != nil {
		return nil, fmt.Errorf("unmarshaling message: %v", err)
	}

	log.Infof("bid %s won in auction %s", win.BidId, win.AuctionId)
	return nil, nil
}

func (s *Service) proposalHandler(from peer.ID, topic string, msg []byte) ([]byte, error) {
	log.Debugf("%s received proposal from %s", topic, from)

	prop := &pb.WinningBidProposal{}
	if err := proto.Unmarshal(msg, prop); err != nil {
		return nil, fmt.Errorf("unmarshaling message: %v", err)
	}
	log.Infof("bid %s received proposal cid %s in auction %s", prop.BidId, prop.ProposalCid, prop.AuctionId)

	// s.store.SetProposalCid()
	return nil, nil
}

func (s *Service) makeBid(auction *pb.Auction, from peer.ID) error {
	if ok := s.filterAuction(auction); !ok {
		log.Infof("not bidding in auction %s from %s", auction.Id, from)
		return nil
	}

	// Get current chain height
	currentEpoch, err := s.fc.GetChainHeight()
	if err != nil {
		return fmt.Errorf("getting chain height: %v", err)
	}

	// Create bids topic
	bids, err := s.peer.NewTopic(s.ctx, broker.BidsTopic(broker.AuctionID(auction.Id)), false)
	if err != nil {
		return fmt.Errorf("creating bids topic: %v", err)
	}
	defer func() { _ = bids.Close() }()
	bids.SetEventHandler(s.eventHandler)

	// Submit bid to auctioneer
	bid := &pb.Bid{
		AuctionId:        auction.Id,
		MinerAddr:        s.bidParams.MinerAddr,
		WalletAddrSig:    s.bidParams.WalletAddrSig,
		AskPrice:         s.bidParams.AskPrice,
		VerifiedAskPrice: s.bidParams.AskPrice,
		StartEpoch:       s.bidParams.DealStartWindow + currentEpoch,
		FastRetrieval:    s.bidParams.FastRetrieval,
	}
	bidj, err := json.MarshalIndent(bid, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling json: %v", err)
	}
	log.Infof("bidding in auction %s from %s: \n%s", auction.Id, from, string(bidj))

	msg, err := proto.Marshal(bid)
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	id, err := bids.Publish(s.ctx, msg, bidsAckTimeout)
	if err != nil {
		return fmt.Errorf("publishing bid: %v", err)
	}
	log.Warn(string(id))

	// Save bid locally
	// if err := s.store.SaveBid(st.Bid{
	// 	ID:               broker.BidID(id),
	// 	AuctionID:        broker.AuctionID(bid.AuctionId),
	// 	AskPrice:         bid.AskPrice,
	// 	VerifiedAskPrice: bid.VerifiedAskPrice,
	// 	StartEpoch:       bid.StartEpoch,
	// 	FastRetrieval:    bid.FastRetrieval,
	// }); err != nil {
	// 	return fmt.Errorf("saving bid: %v", err)
	// }

	log.Debugf("created bid %s in auction %s", id, auction.Id)
	return nil
}

func (s *Service) filterAuction(auction *pb.Auction) bool {
	// Check if auction is still in progress
	if !auction.EndsAt.IsValid() || auction.EndsAt.AsTime().Before(time.Now()) {
		return false
	}

	// Check if deal size is within configured bounds
	if auction.DealSize < s.auctionFilters.DealSize.Min ||
		auction.DealSize > s.auctionFilters.DealSize.Max {
		return false
	}

	// Check if deal duration is within configured bounds
	if auction.DealDuration < s.auctionFilters.DealDuration.Min ||
		auction.DealDuration > s.auctionFilters.DealDuration.Max {
		return false
	}

	return true
}

func (s *Service) downloadDealData(ctx context.Context, proposalCid cid.Cid) error {
	_, err := s.peer.DAGService().Get(ctx, proposalCid)
	if err != nil {
		return fmt.Errorf("getting node %s: %v", proposalCid, err)
	}
	return nil
}
