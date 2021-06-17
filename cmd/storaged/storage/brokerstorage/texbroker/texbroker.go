package texbroker

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	logger "github.com/textileio/go-log/v2"
)

var (
	log = logger.Logger("texbroker")
)

// TODO(jsign): nuke this abstraction and go directly with client.
// TexBroker provides an interface with the Broker subsystem.
type TexBroker struct {
	c *client.Client
}

var _ broker.BrokerRequestor = (*TexBroker)(nil)

// New returns a new TexBroker.
func New(c *client.Client) (*TexBroker, error) {
	b := &TexBroker{
		c: c,
	}

	return b, nil
}

// Create creates a new BrokerRequest.
func (tb *TexBroker) Create(ctx context.Context, c cid.Cid, meta broker.Metadata) (broker.BrokerRequest, error) {
	log.Debugf("creating broker request for cid %s", c)

	br, err := tb.c.Create(ctx, c, meta)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("calling create api: %s", err)
	}
	return br, nil
}

func (tb *TexBroker) CreatePrepared(ctx context.Context, c cid.Cid, meta broker.Metadata, pc broker.PreparedCAR) (broker.BrokerRequest, error) {
	log.Debugf("creating prepared broker request for cid %s", c)

	br, err := tb.c.CreatePrepared(ctx, c, meta, pc)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("calling create prepared api: %s", err)
	}
	return br, nil
}

// Get gets a broker request from its ID.
func (tb *TexBroker) Get(ctx context.Context, id broker.BrokerRequestID) (broker.BrokerRequest, error) {
	br, err := tb.c.Get(ctx, id)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("calling get api: %s", err)
	}
	return br, nil
}
