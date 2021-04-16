package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	core "github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/message/pb"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/marketpeer"
	"google.golang.org/protobuf/proto"
)

var log = golog.Logger("deals/service")

type Config struct {
	RepoPath string
	Peer     marketpeer.Config
}

type Service struct {
	peer *marketpeer.Peer

	ctx       context.Context
	finalizer *finalizer.Finalizer
}

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

	s := &Service{peer: p, ctx: ctx}

	// Subscribe to the global auctions topic
	auctions, err := p.NewTopic(ctx, core.AuctionTopic, true)
	if err != nil {
		return nil, fin.Cleanupf("creating auctions topic: %v", err)
	}
	fin.Add(auctions)
	auctions.SetEventHandler(s.eventHandler)
	auctions.SetMessageHandler(s.auctionsHandler)

	// Subscribe to our own wins topic
	wins, err := p.NewTopic(ctx, core.WinsTopic(p.Self()), true)
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

func (s *Service) Close() error {
	log.Info("service was shutdown")
	return s.finalizer.Cleanup(nil)
}

func (s *Service) Bootstrap() {
	s.peer.Bootstrap()
}

func (s *Service) EnableMDNS(intervalSecs int) error {
	return s.peer.EnableMDNS(intervalSecs)
}

func (s *Service) eventHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s peer event: %s %s", topic, from, msg)
}

func (s *Service) auctionsHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s received auction from %s", topic, from)

	auction := &pb.Auction{}
	if err := proto.Unmarshal(msg, auction); err != nil {
		log.Errorf("unmarshalling message: %v", err)
		return
	}

	go func() {
		if err := s.makeBid(auction); err != nil {
			log.Errorf("making bid: %v", err)
		}
	}()
}

func (s *Service) winsHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s received win from %s", topic, from)

	win := &pb.Win{}
	if err := proto.Unmarshal(msg, win); err != nil {
		log.Errorf("unmarshalling message: %v", err)
		return
	}
	log.Infof("deal won in auction %s with bid %s", win.AuctionId, win.BidId)
}

func (s *Service) makeBid(auction *pb.Auction) error {
	if ok := s.filterAuction(auction); !ok {
		return nil
	}

	// Create bids topic.
	t, err := s.peer.NewTopic(s.ctx, core.BidsTopic(auction.Id), false)
	if err != nil {
		return fmt.Errorf("creating bids topic: %v", err)
	}
	defer t.Close()
	t.SetEventHandler(s.eventHandler)

	// Submit bid to auctioneer.
	msg, err := proto.Marshal(&pb.Bid{
		AuctionId: auction.Id,

		// @todo: Figure out what this should really look like.
		Amount: int64(rand.Intn(100)),
	})
	if err := t.Publish(s.ctx, msg); err != nil {
		return fmt.Errorf("publishing bid: %v", err)
	}
	return nil
}

func (s *Service) filterAuction(auction *pb.Auction) bool {
	if auction.EndsAt.IsValid() && auction.EndsAt.AsTime().After(time.Now()) {
		// Bid on them all, yolo
		return true
	}
	return false
}
