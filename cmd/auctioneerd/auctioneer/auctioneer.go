package auctioneer

// TODO: Add ACK response to incoming bids.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	core "github.com/textileio/broker-core/auctioneer"
	q "github.com/textileio/broker-core/cmd/auctioneerd/queue"
	"github.com/textileio/broker-core/dshelper/txndswrap"
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1/message"
	"github.com/textileio/broker-core/marketpeer"
	"github.com/textileio/broker-core/pubsub"
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
	auctionConf AuctionConfig

	peer     *marketpeer.Peer
	auctions *pubsub.Topic
	bids     map[string]chan core.Bid

	finalizer *finalizer.Finalizer
	lk        sync.Mutex
}

// New returns a new Auctioneer.
func New(peer *marketpeer.Peer, store txndswrap.TxnDatastore, auctionConf AuctionConfig) (*Auctioneer, error) {
	auctionConf, err := setDefaults(auctionConf)
	if err != nil {
		return nil, fmt.Errorf("setting defaults: %v", err)
	}

	fin := finalizer.NewFinalizer()
	ctx, cancel := context.WithCancel(context.Background())
	fin.Add(finalizer.NewContextCloser(cancel))

	a := &Auctioneer{
		peer:        peer,
		bids:        make(map[string]chan core.Bid),
		auctionConf: auctionConf,
	}

	// Create the global auctions topic
	auctions, err := peer.NewTopic(ctx, core.AuctionTopic, false)
	if err != nil {
		return nil, fin.Cleanupf("creating auctions topic: %v", err)
	}
	fin.Add(auctions)
	auctions.SetEventHandler(a.eventHandler)
	a.auctions = auctions

	queue, err := q.NewQueue(store, a.runAuction)
	if err != nil {
		return nil, fin.Cleanupf("creating queue: %v", err)
	}
	fin.Add(queue)
	a.queue = queue

	a.finalizer = fin
	return a, nil
}

func setDefaults(c AuctionConfig) (AuctionConfig, error) {
	if c.Duration <= 0 {
		return c, fmt.Errorf("duration must be greater than zero")
	} else if c.Duration > maxAuctionDuration {
		return c, fmt.Errorf("duration must be less than or equal to %v", maxAuctionDuration)
	}
	return c, nil
}

// Close the auctioneer.
func (a *Auctioneer) Close() error {
	return a.finalizer.Cleanup(nil)
}

// Bootstrap the market peer against well-known network peers.
func (a *Auctioneer) Bootstrap() {
	a.peer.Bootstrap()
}

// EnableMDNS enables an MDNS discovery service.
// This is useful on a local network (testing).
func (a *Auctioneer) EnableMDNS(intervalSecs int) error {
	return a.peer.EnableMDNS(intervalSecs)
}

// CreateAuction creates a new auction.
// New auctions are queud if the auctioneer is busy.
func (a *Auctioneer) CreateAuction() (string, error) {
	id, err := a.queue.CreateAuction(a.auctionConf.Duration)
	if err != nil {
		return "", fmt.Errorf("creating auction: %v", err)
	}

	log.Debugf("created auction %s", id)
	return id, nil
}

// GetAuction returns an auction by id.
func (a *Auctioneer) GetAuction(id string) (*core.Auction, error) {
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
	deadline := auction.StartedAt.Add(time.Duration(auction.Duration))

	// Publish the auction.
	msg, err := proto.Marshal(&pb.Auction{
		Id:     auction.ID,
		EndsAt: timestamppb.New(deadline),
	})
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	if err := a.auctions.Publish(ctx, msg); err != nil {
		return fmt.Errorf("publishing auction: %v", err)
	}

	auction.Bids = make(map[string]core.Bid)
	actx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	for {
		select {
		case <-actx.Done():
			if err := a.selectWinner(ctx, auction); err != nil {
				return fmt.Errorf("selecting winner: %v", err)
			}

			log.Debugf("auction %s completed; total bids: %d", auction.ID, len(auction.Bids))
			return nil
		case bid, ok := <-resCh:
			if ok {
				log.Debugf("auction %s received bid from %s for %d", auction.ID, bid.From, bid.Amount)

				id, err := a.queue.NewID(bid.ReceivedAt)
				if err != nil {
					return fmt.Errorf("generating bid id: %v", err)
				}
				auction.Bids[id] = bid
			}
		}
	}
}

func (a *Auctioneer) eventHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s peer event: %s %s", topic, from, msg)
}

func (a *Auctioneer) bidsHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s received bid from %s", topic, from)

	bid := &pb.Bid{}
	if err := proto.Unmarshal(msg, bid); err != nil {
		log.Errorf("unmarshaling message: %v", err)
		return
	}

	a.lk.Lock()
	defer a.lk.Unlock()
	ch, ok := a.bids[bid.AuctionId]
	if ok {
		ch <- core.Bid{
			From:       from,
			Amount:     bid.Amount,
			ReceivedAt: time.Now(),
		}
	}
}

func (a *Auctioneer) selectWinner(ctx context.Context, auction *core.Auction) error {
	var winner peer.ID
	topBid := math.MinInt64
	for k, v := range auction.Bids {
		if int(v.Amount) > topBid {
			topBid = int(v.Amount)
			winner = v.From
			auction.WinningBid = k
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
		AuctionId: auction.ID,
		BidId:     auction.WinningBid,
	})
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	if err := wins.Publish(ctx, msg); err != nil {
		return fmt.Errorf("publishing win: %v", err)
	}
	return nil
}
