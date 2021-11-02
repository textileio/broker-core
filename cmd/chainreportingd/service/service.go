package service

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/chainreportingd/store"
	"github.com/textileio/broker-core/msgbroker"
	logging "github.com/textileio/go-log/v2"
)

var (
	log = logging.Logger("service")
)

// Service collects data from other daemons and runs the API server.
type Service struct {
	store *store.Store
}

var _ msgbroker.FinalizedDealListener = (*Service)(nil)
var _ msgbroker.ReadyToBatchListener = (*Service)(nil)
var _ msgbroker.NewBatchCreatedListener = (*Service)(nil)

// New creates the service.
func New(mb msgbroker.MsgBroker, postgresURI string) (*Service, error) {
	s, err := store.New(postgresURI)
	if err != nil {
		return nil, err
	}
	service := &Service{store: s}
	if err := msgbroker.RegisterHandlers(mb, service); err != nil {
		return nil, fmt.Errorf("registering msgbroker handlers: %s", err)
	}

	return service, nil
}

// OnReadyToBatch .
func (s *Service) OnReadyToBatch(
	ctx context.Context,
	opID msgbroker.OperationID,
	data []msgbroker.ReadyToBatchData,
) error {
	return nil
}

// OnNewBatchCreated .
func (s *Service) OnNewBatchCreated(
	ctx context.Context,
	batchID broker.BatchID,
	batchCid cid.Cid,
	batchSize int64,
	storageRequestIDs []broker.StorageRequestID,
	origin string,
	manifest []byte,
	carURL *url.URL,
) error {
	return nil
}

// OnFinalizedDeal .
func (s *Service) OnFinalizedDeal(
	ctx context.Context,
	opID msgbroker.OperationID,
	finalizedDeal broker.FinalizedDeal,
) error {
	return nil
}
