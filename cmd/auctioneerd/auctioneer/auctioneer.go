package auctioneer

// TODO: Add ACK response to incoming bids.
// TODO: Allow for multiple winners.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	core "github.com/textileio/broker-core/broker"
	q "github.com/textileio/broker-core/cmd/auctioneerd/auctioneer/queue"
	"github.com/textileio/broker-core/dshelper/txndswrap"
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1/message"
	"github.com/textileio/broker-core/marketpeer"
	"github.com/textileio/broker-core/metrics"
	"github.com/textileio/broker-core/pubsub"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	log = golog.Logger("auctioneer")

	// maxAuctionDuration is the max duration an auction can run for.
	maxAuctionDuration = time.Minute * 10

	// ErrNotFound indicates the requested auction was not found.
	ErrNotFound = errors.New("auction not found")

	// ErrAuctionFailed indicates an auction was not successful.
	ErrAuctionFailed = errors.New("auction failed; no acceptable bids")
)

// AuctionConfig defines auction params.
type AuctionConfig struct {
	// Duration auctions will be held for.
	Duration time.Duration
}

// Auctioneer handles deal auctions for a broker.
type Auctioneer struct {
	queue       *q.Queue
	started     bool
	auctionConf AuctionConfig

	peer     *marketpeer.Peer
	fc       FilClient
	auctions *pubsub.Topic
	bids     map[core.AuctionID]chan core.Bid

	broker core.Broker

	finalizer *finalizer.Finalizer
	lk        sync.Mutex

	statLastCreatedAuction    time.Time
	metricNewAuction          metric.Int64Counter
	metricNewFinalizedAuction metric.Int64Counter
	metricNewBid              metric.Int64Counter
	metricLastCreatedAuction  metric.Int64ValueObserver
}

// New returns a new Auctioneer.
func New(
	peer *marketpeer.Peer,
	store txndswrap.TxnDatastore,
	broker core.Broker,
	fc FilClient,
	auctionConf AuctionConfig,
) (*Auctioneer, error) {
	if err := checkConfig(auctionConf); err != nil {
		return nil, fmt.Errorf("checking config: %v", err)
	}

	a := &Auctioneer{
		peer:        peer,
		fc:          fc,
		bids:        make(map[core.AuctionID]chan core.Bid),
		broker:      broker,
		auctionConf: auctionConf,
		finalizer:   finalizer.NewFinalizer(),
	}
	a.initMetrics()

	queue, err := q.NewQueue(store, a.runAuction)
	if err != nil {
		return nil, a.finalizer.Cleanupf("creating queue: %v", err)
	}
	a.finalizer.Add(queue)
	a.queue = queue

	return a, nil
}

func checkConfig(c AuctionConfig) error {
	if c.Duration <= 0 {
		return fmt.Errorf("duration must be greater than zero")
	} else if c.Duration > maxAuctionDuration {
		return fmt.Errorf("duration must be less than or equal to %v", maxAuctionDuration)
	}
	return nil
}

// Close the auctioneer.
func (a *Auctioneer) Close() error {
	return a.finalizer.Cleanup(nil)
}

// Start the deal auction feed.
// If bootstrap is true, the peer will dial the configured bootstrap addresses
// before creating the deal auction feed.
func (a *Auctioneer) Start(bootstrap bool) error {
	a.lk.Lock()
	defer a.lk.Unlock()
	if a.started {
		return nil
	}

	// Bootstrap against configured addresses.
	if bootstrap {
		a.peer.Bootstrap()
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.finalizer.Add(finalizer.NewContextCloser(cancel))

	// Create the global auctions topic
	auctions, err := a.peer.NewTopic(ctx, core.AuctionTopic, false)
	if err != nil {
		return fmt.Errorf("creating auctions topic: %v", err)
	}
	auctions.SetEventHandler(a.eventHandler)
	a.auctions = auctions
	a.finalizer.Add(auctions)

	log.Info("created the deal auction feed")

	a.started = true
	return nil
}

// CreateAuction creates a new auction.
// New auctions are queud if the auctioneer is busy.
func (a *Auctioneer) CreateAuction(
	storageDealID core.StorageDealID,
	dealSize, dealDuration uint64,
) (core.AuctionID, error) {
	id, err := a.queue.CreateAuction(storageDealID, dealSize, dealDuration, a.auctionConf.Duration)
	if err != nil {
		return "", fmt.Errorf("creating auction: %v", err)
	}

	log.Debugf("created auction %s", id)
	a.metricNewAuction.Add(context.Background(), 1)
	a.statLastCreatedAuction = time.Now()

	return id, nil
}

// GetAuction returns an auction by id.
func (a *Auctioneer) GetAuction(id core.AuctionID) (*core.Auction, error) {
	auction, err := a.queue.GetAuction(id)
	if errors.Is(q.ErrNotFound, err) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("getting auction: %v", err)
	}
	return auction, nil
}

func (a *Auctioneer) runAuction(ctx context.Context, auction *core.Auction) error {
	a.lk.Lock()
	if _, ok := a.bids[auction.ID]; ok {
		a.lk.Unlock()
		return fmt.Errorf("auction %s already started", auction.ID)
	}
	resCh := make(chan core.Bid)
	a.bids[auction.ID] = resCh
	a.lk.Unlock()
	defer func() {
		a.lk.Lock()
		delete(a.bids, auction.ID)
		a.lk.Unlock()
	}()
	log.Debugf("auction %s started", auction.ID)

	// Subscribe to bids topic.
	bids, err := a.peer.NewTopic(ctx, core.BidsTopic(auction.ID), true)
	if err != nil {
		return fmt.Errorf("creating bids topic: %v", err)
	}
	defer func() { _ = bids.Close() }()
	bids.SetEventHandler(a.eventHandler)
	bids.SetMessageHandler(a.bidsHandler)

	// Set deadline
	deadline := auction.StartedAt.Add(auction.Duration)

	// Publish the auction.
	msg, err := proto.Marshal(&pb.Auction{
		Id:           string(auction.ID),
		DealSize:     auction.DealSize,
		DealDuration: auction.DealDuration,
		EndsAt:       timestamppb.New(deadline),
	})
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	if err := a.auctions.Publish(ctx, msg); err != nil {
		return fmt.Errorf("publishing auction: %v", err)
	}

	auction.Bids = make(map[core.BidID]core.Bid)
	actx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	for {
		select {
		case <-actx.Done():
			if err := a.selectWinner(ctx, auction); err != nil {
				a.metricNewFinalizedAuction.Add(ctx, 1, metrics.AttrError)
				return fmt.Errorf("selecting winner: %v", err)
			}

			a.metricNewFinalizedAuction.Add(ctx, 1, metrics.AttrOK)
			// TODO: Ensure auction state is persisted before notifying broker?
			auction.Status = core.AuctionStatusEnded
			if err := a.broker.StorageDealAuctioned(ctx, *auction); err != nil {
				return fmt.Errorf("signaling broker: %v", err)
			}

			log.Debugf("auction %s completed; total bids: %d", auction.ID, len(auction.Bids))
			return nil
		case bid, ok := <-resCh:
			if ok {
				log.Debugf("auction %s received bid from %s: %d", auction.ID, bid.BidderID, bid.AskPrice)

				id, err := a.queue.NewID(bid.ReceivedAt)
				if err != nil {
					return fmt.Errorf("generating bid id: %v", err)
				}
				auction.Bids[core.BidID(id)] = bid
				a.metricNewBid.Add(ctx, 1)
			}
		}
	}
}

func (a *Auctioneer) eventHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s peer event: %s %s", topic, from, msg)
	if topic == core.AuctionTopic && string(msg) == "JOINED" {
		a.peer.Host().ConnManager().Protect(from, "auctioneer:<bidder>")
	}
}

func (a *Auctioneer) bidsHandler(from peer.ID, _ string, msg []byte) {
	bid := &pb.Bid{}
	if err := proto.Unmarshal(msg, bid); err != nil {
		log.Errorf("unmarshaling message: %v", err)
		return
	}

	ok, err := a.fc.VerifyBidder(bid.WalletAddrSig, from, bid.MinerAddr)
	if err != nil {
		log.Errorf("verifying miner address: %v", err)
		return
	}
	if !ok {
		log.Warn("invalid miner address or signature")
		return
	}

	a.lk.Lock()
	defer a.lk.Unlock()
	ch, ok := a.bids[core.AuctionID(bid.AuctionId)]
	if ok {
		ch <- core.Bid{
			MinerAddr:        bid.MinerAddr,
			WalletAddrSig:    bid.WalletAddrSig,
			BidderID:         from,
			AskPrice:         bid.AskPrice,
			VerifiedAskPrice: bid.VerifiedAskPrice,
			StartEpoch:       bid.StartEpoch,
			FastRetrieval:    bid.FastRetrieval,
			ReceivedAt:       time.Now(),
		}
	}
}

func (a *Auctioneer) selectWinner(ctx context.Context, auction *core.Auction) error {
	var winner peer.ID
	topBid := math.MaxInt64
	for k, v := range auction.Bids {
		if int(v.AskPrice) < topBid {
			topBid = int(v.AskPrice)
			winner = v.BidderID
			auction.WinningBids = append(auction.WinningBids, k)
		}
	}

	if winner.Validate() != nil {
		return ErrAuctionFailed
	}

	// Create win topic.
	wins, err := a.peer.NewTopic(ctx, core.WinsTopic(winner), false)
	if err != nil {
		return fmt.Errorf("creating win topic: %v", err)
	}
	defer func() { _ = wins.Close() }()
	wins.SetEventHandler(a.eventHandler)

	// Notify winner.
	msg, err := proto.Marshal(&pb.Win{
		AuctionId: string(auction.ID),
		BidId:     string(auction.WinningBids[0]),
	})
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	if err := wins.Publish(ctx, msg); err != nil {
		return fmt.Errorf("publishing win: %v", err)
	}
	return nil
}
