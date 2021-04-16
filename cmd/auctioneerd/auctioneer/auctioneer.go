package auctioneer

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
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1/message"
	kt "github.com/textileio/broker-core/keytransform"
	"github.com/textileio/broker-core/marketpeer"
	"github.com/textileio/broker-core/pubsub"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	log = golog.Logger("auctioneer")

	AuctionDuration = time.Second * 10

	ErrAuctionFailed = errors.New("auction failed; no acceptable bids")
)

type Auctioneer struct {
	queue *q.Queue

	peer     *marketpeer.Peer
	auctions *pubsub.Topic
	bids     map[string]chan core.Bid

	finalizer *finalizer.Finalizer
	lk        sync.Mutex
}

func New(peer *marketpeer.Peer, store kt.TxnDatastoreExtended) (*Auctioneer, error) {
	fin := finalizer.NewFinalizer()
	ctx, cancel := context.WithCancel(context.Background())
	fin.Add(finalizer.NewContextCloser(cancel))

	a := &Auctioneer{peer: peer, bids: make(map[string]chan core.Bid)}

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

func (a *Auctioneer) Close() error {
	return a.finalizer.Cleanup(nil)
}

func (a *Auctioneer) Bootstrap() {
	a.peer.Bootstrap()
}

func (a *Auctioneer) EnableMDNS(intervalSecs int) error {
	return a.peer.EnableMDNS(intervalSecs)
}

func (a *Auctioneer) CreateAuction() (string, error) {
	auction, err := a.queue.CreateAuction(AuctionDuration)
	if err != nil {
		return "", fmt.Errorf("creating auction: %v", err)
	}

	log.Debugf("created auction %s", auction.ID)
	return auction.ID, nil
}

func (a *Auctioneer) GetAuction(id string) (*core.Auction, error) {
	return a.queue.GetAuction(id)
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
	defer bids.Close()
	bids.SetEventHandler(a.eventHandler)
	bids.SetMessageHandler(a.bidsHandler)

	// Set deadline
	deadline := auction.StartedAt.Add(time.Duration(auction.Duration))

	// Publish the auction.
	msg, err := proto.Marshal(&pb.Auction{
		Id:     auction.ID,
		EndsAt: timestamppb.New(deadline),
	})
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
		log.Errorf("unmarshalling message: %v", err)
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
			auction.Winner = k
		}
	}

	if winner.Validate() != nil {
		return ErrAuctionFailed
	}

	// Create win topic.
	t, err := a.peer.NewTopic(ctx, core.WinsTopic(winner), false)
	if err != nil {
		return fmt.Errorf("creating win topic: %v", err)
	}
	defer t.Close()
	t.SetEventHandler(a.eventHandler)

	// Notify winner.
	msg, err := proto.Marshal(&pb.Win{
		AuctionId: auction.ID,
		BidId:     auction.Winner,
	})
	if err := t.Publish(ctx, msg); err != nil {
		return fmt.Errorf("publishing win: %v", err)
	}
	return nil
}
