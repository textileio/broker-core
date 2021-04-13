package broker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/srstore"
	"github.com/textileio/broker-core/dshelper/txndswrap"
	"github.com/textileio/broker-core/packer"
)

var (
	// ErrNotFound is returned when the broker request doesn't exist.
	ErrNotFound = fmt.Errorf("broker request not found")
	// ErrInvalidCid is returned when the Cid is undefined.
	ErrInvalidCid = fmt.Errorf("the cid can't be undefined")
)

// Broker creates and tracks request to store Cids in
// the Filecoin network.
type Broker struct {
	store  *srstore.Store
	packer packer.Packer
}

// New creates a Broker backed by the provdied `ds`.
func New(ds datastore.TxnDatastore, packer packer.Packer) (*Broker, error) {
	store, err := srstore.New(txndswrap.Wrap(ds, "/broker-store"))
	if err != nil {
		return nil, fmt.Errorf("initializing broker request store: %s", err)
	}

	b := &Broker{
		store:  store,
		packer: packer,
	}
	return b, nil
}

var _ broker.Broker = (*Broker)(nil)

// Create creates a new BrokerRequest with the provided Cid and
// Metadata configuration.
func (b *Broker) Create(ctx context.Context, c cid.Cid, meta broker.Metadata) (broker.BrokerRequest, error) {
	if !c.Defined() {
		return broker.BrokerRequest{}, ErrInvalidCid
	}

	if err := meta.Validate(); err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("invalid metadata: %s", err)
	}

	br := broker.BrokerRequest{
		ID:       broker.BrokerRequestID(uuid.New().String()),
		Status:   broker.StatusBatching,
		Metadata: meta,
	}
	if err := b.store.Save(ctx, br); err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("saving broker request in store: %s", err)
	}

	// We notify the Packer that this BrokerRequest is ready to be considered.
	// We'll receive a call to `(*Broker).CreateStorageDeal(...)` which will contain
	// this BrokerRequest, and continue with the bidding process..
	if err := b.packer.ReadyToPack(br); err != nil {
		// TODO: there's room for improvement here. We can mark this broker-request
		// to be retried in signaling the packer, to avoid be orphaned.
		// Under normal circumstances this shouldn't happen.
		// We can simply save BrokerRequest with a "ReadyToBatch", and have some daemon
		// making sure of notifying and then switching to "Batching"; shoudn't be a big deal.
		return broker.BrokerRequest{}, fmt.Errorf("notifying packer of ready broker request: %s", err)
	}

	return br, nil
}

// Get gets a BrokerRequest by id. If doesn't exist, it returns ErrNotFound.
func (b *Broker) Get(ctx context.Context, ID broker.BrokerRequestID) (broker.BrokerRequest, error) {
	br, err := b.store.Get(ctx, ID)
	if err == srstore.ErrNotFound {
		return broker.BrokerRequest{}, ErrNotFound
	}
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("get broker request from store: %s", err)
	}

	return br, nil
}
