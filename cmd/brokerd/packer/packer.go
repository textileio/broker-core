package packer

import (
	"context"

	"github.com/textileio/broker-core/broker"
	packeri "github.com/textileio/broker-core/packer"
)

// Packer provides batching strategies to bundle multiple
// BrokerRequest into a StorageDeal.
type Packer struct {
}

var _ packeri.Packer = (*Packer)(nil)

// New returns a new Packer.
func New() (*Packer, error) {
	p := &Packer{}

	return p, nil
}

// ReadyToPack signals the packer that there's a new BrokerRequest that can be
// considered. Packer will notify the broker async when the final StorageDeal
// that contains this BrokerRequest gets created.
func (p *Packer) ReadyToPack(ctx context.Context, br broker.BrokerRequest) error {
	// TODO: wire gRPC client whenever ready.
	// For the moment, just accepts without error

	return nil
}
