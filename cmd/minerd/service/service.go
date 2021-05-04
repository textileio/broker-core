package service

// TODO: Store bids (bid history).

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1/message"
	"github.com/textileio/broker-core/marketpeer"
	"google.golang.org/protobuf/proto"
)

var log = golog.Logger("miner/service")

// Config defines params for Service configuration.
type Config struct {
	RepoPath       string
	Peer           marketpeer.Config
	BidParams      BidParams
	AuctionFilters AuctionFilters
}

// BidParams defines how bids are made.
type BidParams struct {
	AskPrice int64 // attoFIL per GiB per epoch
}

// AuctionFilters specifies filters used when selecting auctions to bid on.
type AuctionFilters struct {
	DealDuration MinMaxFilter
	DealSize     MinMaxFilter
}

// MinMaxFilter is used to specify a range for an auction filter.
type MinMaxFilter struct {
	Min uint64
	Max uint64
}

// Service is a miner service that subscribes to brokered deals.
type Service struct {
	peer *marketpeer.Peer

	bidParams      BidParams
	auctionFilters AuctionFilters

	ctx       context.Context
	finalizer *finalizer.Finalizer
}

// New returns a new Service.
func New(conf Config) (*Service, error) {
	fin := finalizer.NewFinalizer()
	ctx, cancel := context.WithCancel(context.Background())
	fin.Add(finalizer.NewContextCloser(cancel))

	// Create miner peer
	p, err := marketpeer.New(conf.Peer)
	if err != nil {
		return nil, fin.Cleanupf("creating peer: %v", err)
	}
	fin.Add(p)

	s := &Service{
		peer:           p,
		bidParams:      conf.BidParams,
		auctionFilters: conf.AuctionFilters,
		ctx:            ctx,
	}

	// Subscribe to the global auctions topic
	auctions, err := p.NewTopic(ctx, broker.AuctionTopic, true)
	if err != nil {
		return nil, fin.Cleanupf("creating auctions topic: %v", err)
	}
	fin.Add(auctions)
	auctions.SetEventHandler(s.eventHandler)
	auctions.SetMessageHandler(s.auctionsHandler)

	// Subscribe to our own wins topic
	wins, err := p.NewTopic(ctx, broker.WinsTopic(p.Self()), true)
	if err != nil {
		return nil, fin.Cleanupf("creating wins topic: %v", err)
	}
	fin.Add(wins)
	wins.SetEventHandler(s.eventHandler)
	wins.SetMessageHandler(s.winsHandler)

	log.Info("service started")

	s.finalizer = fin
	return s, nil
}

// Close the service.
func (s *Service) Close() error {
	log.Info("service was shutdown")
	return s.finalizer.Cleanup(nil)
}

// Bootstrap the market peer against well-known network peers.
func (s *Service) Bootstrap() {
	s.peer.Bootstrap()
}

// EnableMDNS enables an MDNS discovery service.
// This is useful on a local network (testing).
func (s *Service) EnableMDNS(intervalSecs int) error {
	return s.peer.EnableMDNS(intervalSecs)
}

// GetSigningMessage returns a message to be signed by a miner address.
func (s *Service) GetSigningMessage() string {
	return hex.EncodeToString([]byte(s.peer.Self()))
}

func (s *Service) eventHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s peer event: %s %s", topic, from, msg)
}

func (s *Service) auctionsHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s received auction from %s", topic, from)

	auction := &pb.Auction{}
	if err := proto.Unmarshal(msg, auction); err != nil {
		log.Errorf("unmarshaling message: %v", err)
		return
	}

	auctionj, err := json.MarshalIndent(auction, "", "  ")
	if err != nil {
		log.Errorf("marshaling json: %v", err)
		return
	}
	log.Infof("found auction %s from %s: \n%s", auction.Id, from, string(auctionj))

	go func() {
		if err := s.makeBid(auction, from); err != nil {
			log.Errorf("making bid: %v", err)
		}
	}()
}

func (s *Service) winsHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s received win from %s", topic, from)

	win := &pb.Win{}
	if err := proto.Unmarshal(msg, win); err != nil {
		log.Errorf("unmarshaling message: %v", err)
		return
	}
	log.Infof("bid %s won in auction %s", win.BidId, win.AuctionId)
}

func (s *Service) makeBid(auction *pb.Auction, from peer.ID) error {
	if ok := s.filterAuction(auction); !ok {
		log.Infof("not bidding in auction %s from %s", auction.Id, from)
		return nil
	}

	// Create bids topic.
	bids, err := s.peer.NewTopic(s.ctx, broker.BidsTopic(broker.AuctionID(auction.Id)), false)
	if err != nil {
		return fmt.Errorf("creating bids topic: %v", err)
	}
	defer func() { _ = bids.Close() }()
	bids.SetEventHandler(s.eventHandler)

	// Submit bid to auctioneer.
	bid := &pb.Bid{
		AuctionId:        auction.Id,
		MinerId:          "f01123", // TODO
		AskPrice:         s.bidParams.AskPrice,
		VerifiedAskPrice: s.bidParams.AskPrice, // TODO
		StartEpoch:       1234567890,           // TODO
		FastRetrieval:    false,                // TODO
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
	if err := bids.Publish(s.ctx, msg); err != nil {
		return fmt.Errorf("publishing bid: %v", err)
	}
	return nil
}

// TODO: Add defaults.
func (s *Service) filterAuction(auction *pb.Auction) bool {
	// Check if auction is still in progress.
	if !auction.EndsAt.IsValid() || auction.EndsAt.AsTime().Before(time.Now()) {
		return false
	}

	// Check if deal size is within configured bounds.
	if auction.DealSize < s.auctionFilters.DealSize.Min ||
		auction.DealSize > s.auctionFilters.DealSize.Max {
		return false
	}

	// Check if deal duration is within configured bounds.
	if auction.DealDuration < s.auctionFilters.DealDuration.Min ||
		auction.DealDuration > s.auctionFilters.DealDuration.Max {
		return false
	}

	return true
}
