package auctioneer

// TODO: Add ACK response to incoming bids.

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/broker-core/broker"
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

	// Attempts that an auction will run before signaling to the broker that it failed.
	// Auctions will continue to run under the following conditions:
	// 1. While deal replication is greater than the number of winning bids.
	// 2. While at least one owner of a winning bid is unreachable.
	// 3. While signaling the broker results in an error.
	Attempts uint32
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
	if err := validateConfig(auctionConf); err != nil {
		return nil, fmt.Errorf("validating config: %v", err)
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

	queue, err := q.NewQueue(store, a.runAuction, a.finalizeAuction, auctionConf.Attempts)
	if err != nil {
		return nil, a.finalizer.Cleanupf("creating queue: %v", err)
	}
	a.finalizer.Add(queue)
	a.queue = queue

	return a, nil
}

func validateConfig(c AuctionConfig) error {
	if c.Duration <= 0 {
		return fmt.Errorf("duration must be greater than zero")
	} else if c.Duration > maxAuctionDuration {
		return fmt.Errorf("duration must be less than or equal to %v", maxAuctionDuration)
	}
	if c.Attempts == 0 {
		return fmt.Errorf("max attempts must be greater than zero")
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
	dealReplication uint32,
	dealVerified bool,
) (core.AuctionID, error) {
	auction := core.Auction{
		StorageDealID:   storageDealID,
		DealSize:        dealSize,
		DealDuration:    dealDuration,
		DealReplication: dealReplication,
		DealVerified:    dealVerified,
		Status:          broker.AuctionStatusUnspecified,
		Duration:        a.auctionConf.Duration,
	}
	id, err := a.queue.CreateAuction(auction)
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

func (a *Auctioneer) runAuction(ctx context.Context, auction *core.Auction, addBid func(bid core.Bid) error) error {
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

	actx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	for {
		select {
		case <-actx.Done():
			log.Debugf(
				"auction %s completed (attempt=%d/%d); total bids: %d/%d",
				auction.ID,
				auction.Attempts,
				a.auctionConf.Attempts,
				len(auction.Bids),
				auction.DealReplication,
			)
			if err := a.selectWinners(ctx, auction); err != nil {
				return fmt.Errorf("selecting winners: %v", err)
			}
			return nil
		case bid, ok := <-resCh:
			if ok {
				var price int64
				if auction.DealVerified {
					price = bid.VerifiedAskPrice
				} else {
					price = bid.AskPrice
				}
				log.Debugf("auction %s received bid from %s: %d", auction.ID, bid.BidderID, price)
				if err := addBid(bid); err != nil {
					log.Errorf("adding bid to auction %s: %v", auction.ID, err)
				}
				a.metricNewBid.Add(ctx, 1)
			}
		}
	}
}

func (a *Auctioneer) finalizeAuction(ctx context.Context, auction core.Auction) error {
	switch auction.Status {
	case broker.AuctionStatusEnded:
		a.metricNewFinalizedAuction.Add(ctx, 1, metrics.AttrOK)
	case broker.AuctionStatusError:
		a.metricNewFinalizedAuction.Add(ctx, 1, metrics.AttrError)
	default:
		return fmt.Errorf("invalid final status: %s", auction.Status)
	}
	if err := a.broker.StorageDealAuctioned(ctx, auction); err != nil {
		return fmt.Errorf("signaling broker: %v", err)
	}
	return nil
}

func (a *Auctioneer) eventHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s peer event: %s %s", topic, from, msg)
	if topic == core.AuctionTopic && string(msg) == "JOINED" {
		a.peer.Host().ConnManager().Protect(from, "auctioneer:<bidder>")
	}
}

func (a *Auctioneer) bidsHandler(from peer.ID, _ string, msg []byte) {
	if from.Validate() != nil {
		log.Warnf("invalid bidder: %s", from)
		return
	}

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

type bid struct {
	ID  core.BidID
	Bid core.Bid
}

func heapifyBids(bids map[core.BidID]core.Bid, dealVerified bool) *BidHeap {
	h := &BidHeap{dealVerified: dealVerified}
	heap.Init(h)
	for id, b := range bids {
		heap.Push(h, bid{ID: id, Bid: b})
	}
	return h
}

// BidHeap is used to efficiently select auction winners.
type BidHeap struct {
	h            []bid
	dealVerified bool
}

// Len returns the length of h.
func (bh *BidHeap) Len() int {
	return len(bh.h)
}

// Less returns true if the value at j is less than the value at i.
func (bh *BidHeap) Less(i, j int) bool {
	if bh.dealVerified {
		return bh.h[i].Bid.VerifiedAskPrice > bh.h[j].Bid.VerifiedAskPrice
	}
	return bh.h[i].Bid.AskPrice > bh.h[j].Bid.AskPrice
}

// Swap index i and j.
func (bh *BidHeap) Swap(i, j int) {
	bh.h[i], bh.h[j] = bh.h[j], bh.h[i]
}

// Push adds x to h.
func (bh *BidHeap) Push(x interface{}) {
	bh.h = append(bh.h, x.(bid))
}

// Pop removes and returns the last element in h.
func (bh *BidHeap) Pop() (x interface{}) {
	x, bh.h = bh.h[len(bh.h)-1], bh.h[:len(bh.h)-1]
	return x
}

func (a *Auctioneer) selectWinners(ctx context.Context, auction *core.Auction) error {
	if len(auction.Bids) == 0 {
		return ErrAuctionFailed
	}
	var winners []peer.ID
	bh := heapifyBids(auction.Bids, auction.DealVerified)

	// Select lowest bids until deal replication is met w/o overwriting past winners
	selectCount := int(auction.DealReplication) - len(auction.WinningBids)
	for i := 0; i < selectCount; i++ {
		b := heap.Pop(bh).(bid)
		winners = append(winners, b.Bid.BidderID)
		auction.WinningBids = append(auction.WinningBids, b.ID)
	}
	if len(winners) == 0 {
		return ErrAuctionFailed
	}

	// Create win topics.
	for _, w := range winners {
		wins, err := a.peer.NewTopic(ctx, core.WinsTopic(w), false)
		if err != nil {
			return fmt.Errorf("creating win topic: %v", err)
		}
		wins.SetEventHandler(a.eventHandler)

		// Notify winner.
		msg, err := proto.Marshal(&pb.Win{
			AuctionId: string(auction.ID),
			BidId:     string(auction.WinningBids[0]),
		})
		if err != nil {
			_ = wins.Close()
			return fmt.Errorf("marshaling message: %v", err)
		}
		if err := wins.Publish(ctx, msg); err != nil {
			_ = wins.Close()
			return fmt.Errorf("publishing win: %v", err)
		}
		_ = wins.Close()
	}
	return nil
}
