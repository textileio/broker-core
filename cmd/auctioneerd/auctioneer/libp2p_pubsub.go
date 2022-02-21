package auctioneer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	pb "github.com/textileio/bidbot/gen/v1"
	"github.com/textileio/bidbot/lib/auction"
	core "github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/cast"
	"github.com/textileio/crypto/asymmetric"
	rpc "github.com/textileio/go-libp2p-pubsub-rpc"
	"github.com/textileio/go-libp2p-pubsub-rpc/finalizer"
	rpcpeer "github.com/textileio/go-libp2p-pubsub-rpc/peer"
	"google.golang.org/protobuf/proto"
)

// CommChannel represents the communication channel with bidbots.
type CommChannel interface {
	PublishAuction(ctx context.Context, id auction.ID, auctionPb *pb.Auction, bidsHandler BidsHandler) error
	PublishWin(ctx context.Context, id auction.ID, bid auction.BidID, bidder peer.ID, sources auction.Sources) error
	PublishProposal(ctx context.Context, id auction.ID, bid auction.BidID, bidder peer.ID, pcid cid.Cid) error
	Start(bootstrap bool) error
	Info() (*rpcpeer.Info, error)
	ListPeers() []peer.ID
	Close() error
}

// BidsHandler handles bids from bidbots.
type BidsHandler func(peer.ID, *pb.Bid) ([]byte, error)

// BidbotEventsHandler handles bidbot events.
type BidbotEventsHandler func(peer.ID, *pb.BidbotEvent)

// Libp2pPubsub communicates with bidbots via libp2p pubsub.
type Libp2pPubsub struct {
	peer                *rpcpeer.Peer
	ctx                 context.Context
	started             bool
	finalizer           *finalizer.Finalizer
	bidbotEventsHandler BidbotEventsHandler
	auctions            *rpc.Topic
	winsTopics          map[peer.ID]*rpc.Topic
	proposalTopics      map[peer.ID]*rpc.Topic
	lkTopics            sync.Mutex
}

// NewLibp2pPubsub creates a communication channel backed by libp2p pubsub.
func NewLibp2pPubsub(ctx context.Context, conf rpcpeer.Config,
	eventsHandler BidbotEventsHandler) (*Libp2pPubsub, error) {
	p, err := rpcpeer.New(conf)
	if err != nil {
		return nil, fmt.Errorf("creating peer: %v", err)
	}
	fin := finalizer.NewFinalizer()
	fin.Add(p)

	return &Libp2pPubsub{
		peer:                p,
		ctx:                 ctx,
		finalizer:           fin,
		bidbotEventsHandler: eventsHandler,
		winsTopics:          make(map[peer.ID]*rpc.Topic),
		proposalTopics:      make(map[peer.ID]*rpc.Topic),
	}, nil
}

// Start creates the deal auction feed.
func (ps *Libp2pPubsub) Start(bootstrap bool) error {
	if ps.started {
		return nil
	}
	// Bootstrap against configured addresses
	if bootstrap {
		ps.peer.Bootstrap()
	}

	// Create the global auctions topic
	auctions, err := ps.peer.NewTopic(ps.ctx, core.Topic, false)
	if err != nil {
		return fmt.Errorf("creating auctions topic: %v", err)
	}
	auctions.SetEventHandler(ps.eventHandler)
	ps.auctions = auctions
	ps.finalizer.Add(auctions)

	bidbotEvents, err := ps.peer.NewTopic(ps.ctx, core.BidbotEventsTopic, true)
	if err != nil {
		return fmt.Errorf("creating bidbot events topic: %v", err)
	}
	bidbotEvents.SetMessageHandler(func(from peer.ID, _ string, msg []byte) ([]byte, error) {
		event := &pb.BidbotEvent{}
		if err := proto.Unmarshal(msg, event); err != nil {
			return nil, fmt.Errorf("unmarshaling message: %v", err)
		}
		ps.bidbotEventsHandler(from, event)
		return nil, nil
	})
	bidbotEvents.SetEventHandler(ps.eventHandler)
	ps.finalizer.Add(bidbotEvents)

	log.Info("created the deal auction feed")

	ps.started = true
	return nil
}

// Info returns the peer info of the channel.
func (ps *Libp2pPubsub) Info() (*rpcpeer.Info, error) {
	return ps.peer.Info()
}

// ListPeers returns the list of connected peers.
func (ps *Libp2pPubsub) ListPeers() []peer.ID {
	return ps.peer.ListPeers()
}

// Close closes the communication channel.
func (ps *Libp2pPubsub) Close() error {
	log.Info("Libp2pPubsub was shutdown")
	return ps.finalizer.Cleanup(nil)
}

// PublishAuction publishs the auction to the deal auction feed until the context expires, and calls the bid handler for
// each bid for the auction.
func (ps *Libp2pPubsub) PublishAuction(ctx context.Context, id auction.ID, auctionPb *pb.Auction,
	bidsHandler BidsHandler) error {
	// Subscribe to bids topic
	topic, err := ps.peer.NewTopic(ctx, core.BidsTopic(id), true)
	if err != nil {
		return fmt.Errorf("creating bids topic: %v", err)
	}
	defer func() {
		if err := topic.Close(); err != nil {
			log.Errorf("closing bids topic: %v", err)
		}
	}()
	topic.SetEventHandler(ps.eventHandler)
	topic.SetMessageHandler(func(from peer.ID, _ string, msg []byte) ([]byte, error) {
		log.Debugf("received bid from peerid %s", from)
		now := time.Now()
		defer func() {
			log.Debugf("processing bid from %s took %dms", from, time.Since(now).Milliseconds())
		}()
		if err := from.Validate(); err != nil {
			return nil, fmt.Errorf("invalid bidder: %v", err)
		}
		pbid := &pb.Bid{}
		if err := proto.Unmarshal(msg, pbid); err != nil {
			return nil, fmt.Errorf("unmarshaling message: %v", err)
		}
		return bidsHandler(from, pbid)
	})
	defer topic.SetMessageHandler(nil)

	// Publish the auction
	msg, err := proto.Marshal(auctionPb)
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	if _, err := ps.auctions.Publish(ctx, msg, rpc.WithRepublishing(true), rpc.WithIgnoreResponse(true)); err != nil {
		return fmt.Errorf("publishing auction: %v", err)
	}
	<-ctx.Done()
	return nil
}

// PublishWin publishs the winning bid message to the specific bidder.
func (ps *Libp2pPubsub) PublishWin(ctx context.Context, id core.ID, bid core.BidID,
	bidder peer.ID, sources core.Sources) error {
	topic, err := ps.winsTopicFor(ctx, bidder)
	if err != nil {
		return fmt.Errorf("creating win topic: %v", err)
	}
	confidential, err := proto.Marshal(&pb.WinningBidConfidential{
		Sources: cast.SourcesToPb(sources),
	})
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	pk, err := bidder.ExtractPublicKey()
	if err != nil {
		return fmt.Errorf("extracting public key from bidder ID: %v", err)
	}
	encryptKey, err := asymmetric.FromPubKey(pk)
	if err != nil {
		return fmt.Errorf("encryption key from public key: %v", err)
	}
	encrypted, err := encryptKey.Encrypt(confidential)
	if err != nil {
		return fmt.Errorf("encrypting: %v", err)
	}
	msg, err := proto.Marshal(&pb.WinningBid{
		AuctionId: string(id),
		BidId:     string(bid),
		Encrypted: encrypted,
	})
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	tctx, cancel := context.WithTimeout(ctx, NotifyTimeout)
	defer cancel()
	res, err := topic.Publish(tctx, msg)
	if err != nil {
		return fmt.Errorf("publishing win to %s in auction %s: %v", bidder, id, err)
	}
	r := <-res
	if errors.Is(r.Err, rpc.ErrResponseNotReceived) {
		return fmt.Errorf("publishing win to %s in auction %s: %v", bidder, id, r.Err)
	} else if r.Err != nil {
		return fmt.Errorf("publishing win in auction %s; bidder %s returned error: %v", id, bidder, r.Err)
	}
	return nil
}

// PublishProposal publishs the proposal to the specific bidder.
func (ps *Libp2pPubsub) PublishProposal(ctx context.Context, id core.ID, bid core.BidID,
	bidder peer.ID, pcid cid.Cid) error {
	topic, err := ps.proposalTopicFor(ctx, bidder)
	if err != nil {
		return fmt.Errorf("creating proposals topic: %v", err)
	}
	msg, err := proto.Marshal(&pb.WinningBidProposal{
		AuctionId:   string(id),
		BidId:       string(bid),
		ProposalCid: pcid.String(),
	})
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	tctx, cancel := context.WithTimeout(ctx, NotifyTimeout)
	defer cancel()
	res, err := topic.Publish(tctx, msg, rpc.WithRepublishing(true))
	if err != nil {
		return err
	}
	r := <-res
	if errors.Is(r.Err, rpc.ErrResponseNotReceived) {
		return err
	} else if r.Err != nil {
		return fmt.Errorf("bidder returned error: %v", r.Err)
	}
	return nil
}

func (ps *Libp2pPubsub) winsTopicFor(ctx context.Context, peer peer.ID) (*rpc.Topic, error) {
	ps.lkTopics.Lock()
	topic, exists := ps.winsTopics[peer]
	if exists {
		ps.lkTopics.Unlock()
		return topic, nil
	}
	defer ps.lkTopics.Unlock()
	topic, err := ps.peer.NewTopic(ctx, core.WinsTopic(peer), false)
	if err != nil {
		return nil, err
	}
	topic.SetEventHandler(ps.eventHandler)
	ps.winsTopics[peer] = topic
	ps.finalizer.Add(topic)
	return topic, nil
}

func (ps *Libp2pPubsub) proposalTopicFor(ctx context.Context, peer peer.ID) (*rpc.Topic, error) {
	ps.lkTopics.Lock()
	topic, exists := ps.proposalTopics[peer]
	if exists {
		ps.lkTopics.Unlock()
		return topic, nil
	}
	defer ps.lkTopics.Unlock()
	topic, err := ps.peer.NewTopic(ctx, core.ProposalsTopic(peer), false)
	if err != nil {
		return nil, err
	}
	topic.SetEventHandler(ps.eventHandler)
	ps.proposalTopics[peer] = topic
	ps.finalizer.Add(topic)
	return topic, nil
}

func (ps *Libp2pPubsub) eventHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s peer event: %s %s", topic, from, msg)
	if topic == core.Topic && string(msg) == "JOINED" {
		ps.peer.Host().ConnManager().Protect(from, "auctioneer:<bidder>")
	}
}
