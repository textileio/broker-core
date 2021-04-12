package texbroker

import (
	"context"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

type TexBroker struct {
}

var _ broker.Broker = (*TexBroker)(nil)

func New() (*TexBroker, error) {
	return &TexBroker{}, nil
}

func (tb *TexBroker) Create(ctx context.Context, c cid.Cid, meta broker.Metadata) (broker.BrokerRequest, error) {
	// TODO: Make the implementation once we have the Broker API to call.
	// For now, just fake it.

	return broker.BrokerRequest{
		ID:     uuid.New().String(),
		Status: broker.StatusBatching,
	}, nil

}
