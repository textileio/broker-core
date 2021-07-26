package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/finalizer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/piecerd/piecer"
	"github.com/textileio/broker-core/cmd/piecerd/store"
	mbroker "github.com/textileio/broker-core/msgbroker"
	golog "github.com/textileio/go-log/v2"
)

var log = golog.Logger("piecer/service")

// Config defines params for Service configuration.
type Config struct {
	PostgresURI    string
	IpfsMultiaddrs []multiaddr.Multiaddr

	DaemonFrequency time.Duration
	RetryDelay      time.Duration
}

// Service is a gRPC service wrapper around a piecer.
type Service struct {
	mb        mbroker.MsgBroker
	piecer    *piecer.Piecer
	finalizer *finalizer.Finalizer
}

var _ mbroker.NewBatchCreatedListener = (*Service)(nil)

// New returns a new Service.
func New(mb mbroker.MsgBroker, conf Config) (*Service, error) {
	if err := validateConfig(conf); err != nil {
		return nil, fmt.Errorf("config is invalid: %s", err)
	}
	fin := finalizer.NewFinalizer()

	lib, err := piecer.New(conf.PostgresURI, conf.IpfsMultiaddrs, mb, conf.DaemonFrequency, conf.RetryDelay)
	if err != nil {
		return nil, fin.Cleanupf("creating piecer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		mb:        mb,
		piecer:    lib,
		finalizer: fin,
	}

	if err := mbroker.RegisterHandlers(mb, s); err != nil {
		return nil, fmt.Errorf("registering msgbroker handlers: %s", err)
	}

	return s, nil
}

// OnNewBatchCreated handles messages for new-batch-created topic.
func (s *Service) OnNewBatchCreated(
	ctx context.Context,
	batchID broker.BatchID,
	batchCid cid.Cid,
	_ []broker.StorageRequestID) error {
	err := s.piecer.ReadyToPrepare(ctx, batchID, batchCid)
	if errors.Is(err, store.ErrBatchExists) {
		log.Warnf("batch-id %s batch-cid %s already processed, acking", batchID, batchCid)
		return nil
	}
	if err != nil {
		return fmt.Errorf("queuing data-cid to be prepared: %s", err)
	}

	return nil
}

// Close the service.
func (s *Service) Close() error {
	defer log.Info("service was shutdown")

	return s.finalizer.Cleanup(nil)
}

func validateConfig(conf Config) error {
	if len(conf.IpfsMultiaddrs) == 0 {
		return fmt.Errorf("ipfs multiaddr list is empty")
	}
	if conf.DaemonFrequency == 0 {
		return fmt.Errorf("daemon frequency is zero")
	}
	if conf.RetryDelay == 0 {
		return fmt.Errorf("retry delay is zero")
	}
	if conf.PostgresURI == "" {
		return errors.New("postgres uri is empty")
	}

	return nil
}
