package texbroker

import (
	"context"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/cmd/uploaderd/storage"
	"github.com/textileio/broker-core/cmd/uploaderd/storage/brokerstorage/broker"
)

type TexBroker struct {
}

var _ broker.Broker = (*TexBroker)(nil)

func New() (*TexBroker, error) {
	return &TexBroker{}, nil
}

func (tb *TexBroker) CreateStorageRequest(ctx context.Context, c cid.Cid, meta storage.Metadata) (storage.StorageRequest, error) {
	// TODO: Make the implementation once we have the Broker API to call.
	// For now, just fake it.

	return storage.StorageRequest{
		ID:         uuid.New().String(),
		StatusCode: storage.StorageRequestStatusPendingPrepare,
	}, nil

}
