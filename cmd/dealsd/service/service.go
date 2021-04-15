package service

import (
	"context"
	"fmt"

	golog "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/peer"
	"github.com/textileio/broker-core/pubsub"
)

const (
	LogName = "deals/service"
)

var (
	log = golog.Logger(LogName)
)

type Config struct {
	Peer peer.Config
}

type Service struct {
	peer      *peer.Peer
	deals     *pubsub.Topic
	finalizer *finalizer.Finalizer
}

func New(conf Config) (*Service, error) {
	p, err := peer.New(conf.Peer)
	if err != nil {
		return nil, fmt.Errorf("creating peer: %v", err)
	}

	var fin = finalizer.NewFinalizer()
	ctx, cancel := context.WithCancel(context.Background())
	fin.Add(finalizer.NewContextCloser(cancel))

	// Subscribe to market deals
	deals, err := p.NewTopic(ctx, string(auctioneer.ProtocolDeals), true)
	if err != nil {
		return nil, fin.Cleanupf("creating deals topic: %v", err)
	}

	s := &Service{
		peer:      p,
		deals:     deals,
		finalizer: fin,
	}

	// deals.SetEventHandler(s.eventHandler)
	// deals.SetMessageHandler(s.messageHandler)

	return s, nil
}

func (s *Service) Close() error {
	return s.finalizer.Cleanup(nil)
}
