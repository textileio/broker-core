package service

import (
	"context"
	"fmt"
	"sync"

	golog "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/pb"
	"github.com/textileio/broker-core/finalizer"
	kt "github.com/textileio/broker-core/keytransform"
	"github.com/textileio/broker-core/peer"
	"github.com/textileio/broker-core/pubsub"
	"github.com/textileio/broker-core/sempool"
)

const (
	LogName = "auctioneer/service"
)

var (
	log = golog.Logger(LogName)
)

type Config struct {
	Peer peer.Config
}

type Service struct {
	pb.UnimplementedAPIServiceServer

	peer  *peer.Peer
	store *kt.TxnDatastoreExtended
	topic *pubsub.Topic

	semaphores *sempool.SemaphorePool
	lk         sync.Mutex
	finalizer  *finalizer.Finalizer
}

var _ pb.APIServiceServer = (*Service)(nil)

func New(conf Config) (*Service, error) {
	p, err := peer.New(conf.Peer)
	if err != nil {
		return nil, fmt.Errorf("creating peer: %v", err)
	}

	var fin = finalizer.NewFinalizer()
	ctx, cancel := context.WithCancel(context.Background())
	fin.Add(finalizer.NewContextCloser(cancel))

	// Subscribe to market deals
	topic, err := p.NewTopic(ctx, string(auctioneer.ProtocolDeals), false)
	if err != nil {
		return nil, fin.Cleanupf("creating deals topic: %v", err)
	}

	s := &Service{
		peer:      p,
		topic:     topic,
		finalizer: fin,
	}

	// deals.SetEventHandler(s.eventHandler)
	// deals.SetMessageHandler(s.messageHandler)

	return s, nil
}

func (s *Service) Close() error {
	return s.finalizer.Cleanup(nil)
}

func (s *Service) CreateAuction(ctx context.Context, req *pb.CreateAuctionRequest) (*pb.CreateAuctionResponse, error) {
	return nil, nil
}
