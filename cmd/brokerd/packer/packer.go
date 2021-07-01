package packer

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/packerd/client"
	packeri "github.com/textileio/broker-core/packer"
	"github.com/textileio/broker-core/rpc"
)

// Packer provides batching strategies to bundle multiple
// BrokerRequest into a StorageDeal.
type Packer struct {
	c *client.Client
}

var _ packeri.Packer = (*Packer)(nil)

// New returns a new Packer.
func New(addr string) (*Packer, error) {
	c, err := client.NewClient(addr, rpc.GetClientOpts(addr)...)
	if err != nil {
		return nil, fmt.Errorf("creating client: %v", err)
	}
	return &Packer{c: c}, nil
}

// ReadyToPack signals the packer that there's a new BrokerRequest that can be
// considered. Packer will notify the broker async when the final StorageDeal
// that contains this BrokerRequest gets created.
func (p *Packer) ReadyToPack(ctx context.Context, id broker.BrokerRequestID, dataCid cid.Cid) error {
	if err := p.c.ReadyToPack(ctx, id, dataCid); err != nil {
		return fmt.Errorf("ready to pack client: %s", err)
	}

	return nil
}

// Close closes the packer.
func (p *Packer) Close() error {
	if err := p.c.Close(); err != nil {
		return fmt.Errorf("closing gRPC client: %s", err)
	}
	return nil
}
