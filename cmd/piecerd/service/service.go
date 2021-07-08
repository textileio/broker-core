package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/dshelper/txndswrap"
	"github.com/textileio/bidbot/lib/finalizer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/piecerd/piecer"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	mbroker "github.com/textileio/broker-core/msgbroker"
	golog "github.com/textileio/go-log/v2"
	"google.golang.org/protobuf/proto"
)

var log = golog.Logger("piecer/service")

// Config defines params for Service configuration.
type Config struct {
	Listener net.Listener

	IpfsMultiaddrs []multiaddr.Multiaddr
	Datastore      txndswrap.TxnDatastore

	DaemonFrequency time.Duration
	RetryDelay      time.Duration
}

// Service is a gRPC service wrapper around a piecer.
type Service struct {
	mb        mbroker.MsgBroker
	piecer    *piecer.Piecer
	finalizer *finalizer.Finalizer
}

// New returns a new Service.
func New(mb mbroker.MsgBroker, conf Config) (*Service, error) {
	fin := finalizer.NewFinalizer()

	lib, err := piecer.New(conf.Datastore, conf.IpfsMultiaddrs, mb, conf.DaemonFrequency, conf.RetryDelay)
	if err != nil {
		return nil, fin.Cleanupf("creating piecer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		mb:        mb,
		piecer:    lib,
		finalizer: fin,
	}

	mb.RegisterTopicHandler("piecer-new-batch-created", "new-batch-created", s.newBatchCreatedHandler)

	return s, nil
}

func (s *Service) newBatchCreatedHandler(data []byte) error {
	r := &pb.NewBatchCreated{}
	if err := proto.Unmarshal(data, r); err != nil {
		return fmt.Errorf("unmarshal ready to prepare request: %s", err)
	}

	if r.Id == "" {
		return errors.New("storage deal id is empty")
	}

	batchCid, err := cid.Cast(r.BatchCid)
	if err != nil {
		return fmt.Errorf("decoding batch cid: %s", err)
	}

	if !batchCid.Defined() {
		return errors.New("data cid is undefined")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := s.piecer.ReadyToPrepare(ctx, broker.StorageDealID(r.Id), batchCid); err != nil {
		return fmt.Errorf("queuing data-cid to be prepared: %s", err)
	}

	return nil
}

// Close the service.
func (s *Service) Close() error {
	defer log.Info("service was shutdown")

	return s.finalizer.Cleanup(nil)
}
