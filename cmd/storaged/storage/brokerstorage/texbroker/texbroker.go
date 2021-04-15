package texbroker

import (
	"context"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

// TexBroker provides an interface with the Broker subsystem.
type TexBroker struct {
}

var _ broker.Broker = (*TexBroker)(nil)

// New returns a new TexBroker.
func New() (*TexBroker, error) {
	return &TexBroker{}, nil
}

// Create creates a new BrokerRequest.
func (tb *TexBroker) Create(ctx context.Context, c cid.Cid, meta broker.Metadata) (broker.BrokerRequest, error) {
	// TODO: Make the implementation once we have the Broker API to call.
	// For now, just fake it.

	return broker.BrokerRequest{
		ID:     broker.BrokerRequestID(uuid.New().String()),
		Status: broker.BrokerRequestBatching,
	}, nil
}

// Get gets a broker request from its ID.
// TODO: whenever th gRPC is wired, it needs to detect and document
// the "not-found" case.
func (tb *TexBroker) Get(ctx context.Context, id broker.BrokerRequestID) (broker.BrokerRequest, error) {
	// TODO: Make the implementation once we have the Broker API to call.
	// For now, just fake it.

	return broker.BrokerRequest{
		ID:     id,
		Status: broker.BrokerRequestUnknown,
	}, nil
}
