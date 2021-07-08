package service

import (
	"context"
	"net"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/dshelper/txndswrap"
	"github.com/textileio/bidbot/lib/finalizer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/piecerd/piecer"
	pb "github.com/textileio/broker-core/gen/broker/piecer/v1"
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

	mb.RegisterTopicHandler("piecer-new-batch-created", "new-batch-created", s.readyToPrepareHandler)

	return s, nil
}

func (s *Service) readyToPrepareHandler(data []byte, ack mbroker.AckMessageFunc, nack mbroker.NackMessageFunc) {
	r := &pb.ReadyToPrepareRequest{}
	if err := proto.Unmarshal(data, r); err != nil {
		log.Errorf("unmarshal ready to prepare request: %s", err)
		nack()
		return
	}

	if r.StorageDealId == "" {
		log.Error("storage deal id is empty")
		nack()
		return
	}

	dataCid, err := cid.Decode(r.DataCid)
	if err != nil {
		log.Errorf("decoding data cid: %s", err)
		nack()
		return
	}

	if !dataCid.Defined() {
		log.Error("data cid is undefined")
		nack()
		return
	}

	// TODO(jsign): remove
	ctx := context.Background()
	if err := s.piecer.ReadyToPrepare(ctx, broker.StorageDealID(r.StorageDealId), dataCid); err != nil {
		log.Errorf("queuing data-cid to be prepared: %s", err)
		nack()
		return
	}

	ack()
}

// Close the service.
func (s *Service) Close() error {
	defer log.Info("service was shutdown")

	return s.finalizer.Cleanup(nil)
}
