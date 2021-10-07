package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/cmd/packerd/packer"
	"github.com/textileio/broker-core/cmd/packerd/store"
	"github.com/textileio/broker-core/msgbroker"
	mbroker "github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/go-libp2p-pubsub-rpc/finalizer"
	golog "github.com/textileio/go-log/v2"
)

var log = golog.Logger("packer/service")

// Config defines params for Service configuration.
type Config struct {
	PostgresURI string

	PinnerMultiaddr string
	IpfsMaddrs      []multiaddr.Multiaddr

	DaemonFrequency        time.Duration
	ExportMetricsFrequency time.Duration

	TargetSectorSize       int64
	BatchMinSize           int64
	BatchMinWaiting        time.Duration
	BatchWaitScalingFactor int64

	CARUploader  packer.CARUploader
	CARExportURL string
}

// Service is a gRPC service wrapper around an packer.
type Service struct {
	packer    *packer.Packer
	finalizer *finalizer.Finalizer
}

var _ mbroker.ReadyToBatchListener = (*Service)(nil)

// New returns a new Service.
func New(mb mbroker.MsgBroker, conf Config) (*Service, error) {
	if err := validateConfig(conf); err != nil {
		return nil, fmt.Errorf("config is invalid: %s", err)
	}

	fin := finalizer.NewFinalizer()

	ma, err := multiaddr.NewMultiaddr(conf.PinnerMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("parsing ipfs client multiaddr: %s", err)
	}
	pinnerClient, err := httpapi.NewApi(ma)
	if err != nil {
		return nil, fmt.Errorf("creating ipfs client: %s", err)
	}
	opts := []packer.Option{
		packer.WithDaemonFrequency(conf.DaemonFrequency),
		packer.WithSectorSize(conf.TargetSectorSize),
		packer.WithCARExportURL(conf.CARExportURL),
		packer.WithBatchMinSize(conf.BatchMinSize),
		packer.WithBatchMinWaiting(conf.BatchMinWaiting),
		packer.WithBatchWaitScalingFactor(conf.BatchWaitScalingFactor),
		packer.WithCARUploader(conf.CARUploader),
	}

	lib, err := packer.New(conf.PostgresURI, pinnerClient, conf.IpfsMaddrs, mb, opts...)
	if err != nil {
		return nil, fin.Cleanupf("creating packer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		packer:    lib,
		finalizer: fin,
	}

	if err := mbroker.RegisterHandlers(mb, s, msgbroker.WithACKDeadline(time.Minute*5)); err != nil {
		return nil, fmt.Errorf("registering msgbroker handlers: %s", err)
	}

	return s, nil
}

// OnReadyToBatch process a message for data ready to be included in a batch.
func (s *Service) OnReadyToBatch(ctx context.Context, opID mbroker.OperationID, srs []mbroker.ReadyToBatchData) error {
	err := s.packer.ReadyToBatch(ctx, opID, srs)
	if errors.Is(err, store.ErrOperationIDExists) {
		log.Warnf("operation-id %s already processed, acking", opID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("processing ready to batch: %s", err)
	}

	return nil
}

// Close the service.
func (s *Service) Close() error {
	log.Info("closing service")
	defer log.Info("service was shutdown")

	return s.finalizer.Cleanup(nil)
}

func validateConfig(conf Config) error {
	if conf.PinnerMultiaddr == "" {
		return fmt.Errorf("ipfs pinner multiaddr is empty")
	}
	if conf.PostgresURI == "" {
		return fmt.Errorf("postgres uri is empty")
	}

	return nil
}
