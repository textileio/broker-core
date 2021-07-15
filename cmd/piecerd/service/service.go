package service

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/piecerd/piecer"
	mbroker "github.com/textileio/broker-core/msgbroker"
	golog "github.com/textileio/go-log/v2"
)

var log = golog.Logger("piecer/service")

// Config defines params for Service configuration.
type Config struct {
	Listener net.Listener

	IpfsMultiaddrs []multiaddr.Multiaddr
	AckDeadline    time.Duration
}

// Service is a gRPC service wrapper around a piecer.
type Service struct {
	mb     mbroker.MsgBroker
	piecer *piecer.Piecer
}

// New returns a new Service.
func New(mb mbroker.MsgBroker, conf Config) (*Service, error) {
	lib, err := piecer.New(conf.IpfsMultiaddrs, mb)
	if err != nil {
		return nil, fmt.Errorf("creating piecer: %v", err)
	}

	s := &Service{
		mb:     mb,
		piecer: lib,
	}

	if err := mbroker.RegisterHandlers(mb, &s, mbroker.WithACKDeadline(conf.AckDeadline)); err != nil {
		return nil, fmt.Errorf("registering msgbroker handlers: %s", err)
	}

	return s, nil
}

// OnNewBatchCreated handles messages for new-batch-created topic.
func (s *Service) OnNewBatchCreated(sdID broker.StorageDealID, batchCid cid.Cid) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := s.piecer.ReadyToPrepare(ctx, sdID, batchCid); err != nil {
		return fmt.Errorf("queuing data-cid to be prepared: %s", err)
	}

	return nil
}
