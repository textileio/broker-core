package piecer

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

type Piecer struct {
}

func New() (*Piecer, error) {
	p := &Piecer{}

	return p, nil
}

func (p *Piecer) ReadyToPrepare(ctx context.Context, id broker.StorageDealID, c cid.Cid) error {
	// TODO: wire with gRPC API when available.

	return nil
}
