package piecer

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

// Piecer provides a data-preparing pipeline for StorageDeals.
type Piecer interface {
	// ReadyToPrepare signals the Piecer that a new StorageDeal is ready to be
	// prepared. When that happens, the Piecer calls the broker async to report
	// the result, so it can start with the auction.
	ReadyToPrepare(ctx context.Context, id broker.StorageDealID, c cid.Cid) error
}
