package packer

import (
	"context"

	"github.com/textileio/broker-core/broker"
	packeri "github.com/textileio/broker-core/packer"
)

type Packer struct {
}

var _ packeri.Packer = (*Packer)(nil)

func New() (*Packer, error) {
	p := &Packer{}

	return p, nil
}

func (p *Packer) ReadyToPack(ctx context.Context, br broker.BrokerRequest) error {
	// TODO: wire gRPC client whenever ready.
	// For the moment, just accepts without error

	return nil
}
