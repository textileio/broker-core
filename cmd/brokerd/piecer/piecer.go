package piecer

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

// Piecer provides a data-preparation pipeline for StorageDeals.
type Piecer struct {
}

// New returns a nice Piecer.
func New() (*Piecer, error) {
	p := &Piecer{}

	return p, nil
}

// ReadyToPrepare signals the Piecer that a new StorageDeal is ready to be prepared.
// Piecer will call the broker async with the end result.
func (p *Piecer) ReadyToPrepare(ctx context.Context, id broker.StorageDealID, c cid.Cid) error {
	// TODO: wire with gRPC API when available.

	return nil
}
