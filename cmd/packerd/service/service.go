package service

import (
	"context"
	"fmt"
	"time"

	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/dshelper"
	"github.com/textileio/bidbot/lib/finalizer"
	"github.com/textileio/broker-core/cmd/packerd/packer"
	mbroker "github.com/textileio/broker-core/msgbroker"
	golog "github.com/textileio/go-log/v2"
)

var log = golog.Logger("packer/service")

// Config defines params for Service configuration.
type Config struct {
	MongoDBName string
	MongoURI    string

	IpfsAPIMultiaddr string

	DaemonFrequency        time.Duration
	ExportMetricsFrequency time.Duration

	TargetSectorSize int64
	BatchMinSize     uint
}

// Service is a gRPC service wrapper around an packer.
type Service struct {
	packer    *packer.Packer
	finalizer *finalizer.Finalizer
}

// New returns a new Service.
func New(mb mbroker.MsgBroker, conf Config) (*Service, error) {
	if err := validateConfig(conf); err != nil {
		return nil, fmt.Errorf("config is invalid: %s", err)
	}

	fin := finalizer.NewFinalizer()

	ds, err := dshelper.NewMongoTxnDatastore(conf.MongoURI, conf.MongoDBName)
	if err != nil {
		return nil, fmt.Errorf("creating datastore: %s", err)
	}
	fin.Add(ds)

	ma, err := multiaddr.NewMultiaddr(conf.IpfsAPIMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("parsing ipfs client multiaddr: %s", err)
	}
	ipfsClient, err := httpapi.NewApi(ma)
	if err != nil {
		return nil, fmt.Errorf("creating ipfs client: %s", err)
	}
	opts := []packer.Option{
		packer.WithDaemonFrequency(conf.DaemonFrequency),
		packer.WithSectorSize(conf.TargetSectorSize),
		packer.WithBatchMinSize(conf.BatchMinSize),
	}

	lib, err := packer.New(ds, ipfsClient, mb, opts...)
	if err != nil {
		return nil, fin.Cleanupf("creating packer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		packer:    lib,
		finalizer: fin,
	}

	if err := mbroker.RegisterHandlers(mb, s); err != nil {
		return nil, fmt.Errorf("registering msgbroker handlers: %s", err)
	}

	return s, nil
}

// OnReadyToBatch process a message for data ready to be included in a batch.
func (s *Service) OnReadyToBatch(ctx context.Context, readyDataCids []mbroker.ReadyToBatchData) error {
	for _, rdc := range readyDataCids {
		if err := s.packer.ReadyToBatch(ctx, rdc.BrokerRequestID, rdc.DataCid); err != nil {
			return fmt.Errorf("queuing broker request: %s", err)
		}
	}

	return nil
}

// Close the service.
func (s *Service) Close() error {
	defer log.Info("service was shutdown")

	return s.finalizer.Cleanup(nil)
}

func validateConfig(conf Config) error {
	if conf.IpfsAPIMultiaddr == "" {
		return fmt.Errorf("ipfs api multiaddr is empty")
	}
	if conf.MongoDBName == "" {
		return fmt.Errorf("mongo db name is empty")
	}
	if conf.MongoURI == "" {
		return fmt.Errorf("mongo uri is empty")
	}

	return nil
}
