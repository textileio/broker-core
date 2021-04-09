package broker

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/cmd/storaged/storage"
)

type Broker interface {
	CreateStorageRequest(ctx context.Context, c cid.Cid, meta storage.Metadata) (storage.StorageRequest, error)
}
