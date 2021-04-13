package brocker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/srstore"
	"github.com/textileio/broker-core/dshelper/txndswrap"
)

// Broker creates and tracks request to store Cids in
// the Filecoin network.
type Broker struct {
	store *srstore.Store
}

// New creates a Broker backed by the provdied `ds`.
func New(ds datastore.TxnDatastore) (*Broker, error) {
	store, err := srstore.New(txndswrap.Wrap(ds, "/broker-store"))
	if err != nil {
		return nil, fmt.Errorf("initializing broker request store: %s", err)
	}

	b := &Broker{
		store: store,
	}
	return b, nil
}

var _ broker.Broker = (*Broker)(nil)

// Create creates a new BrokerRequest with the provided Cid and
// Metadata configuration.
func (b *Broker) Create(ctx context.Context, c cid.Cid, meta broker.Metadata) (broker.BrokerRequest, error) {
	if !c.Defined() {
		return broker.BrokerRequest{}, fmt.Errorf("cid is undefined")
	}
	if err := validMetadata(meta); err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("invalid metadata: %s", err)
	}

	br := broker.BrokerRequest{
		ID:       uuid.New().String(),
		Status:   broker.StatusIdle,
		Metadata: meta,
	}
	if err := b.store.Save(ctx, br); err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("saving broker request in store: %s", err)
	}

	// The `Idle` broker request will be detected
	// by `batcherd` to continue the process.

	return br, nil
}

// TODO: move to other place.
func validMetadata(meta broker.Metadata) error {
	// TODO: place logic here to validate
	// metadata attributes. e.g: `region` is valid
	// if isn't empty.
	return nil
}
