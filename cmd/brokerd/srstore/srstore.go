package srstore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/textileio/broker-core/broker"
)

var (
	// ErrNotFound is returned if the broker request doesn't exist.
	ErrNotFound = fmt.Errorf("broker request not found")

	// Namespace "/broker-request/{id}" contains
	// the current `BrokerRequest` data for an `id`.
	prefixBrokerRequest = datastore.NewKey("broker-request")
)

// Store provides a persistent layer for broker requests.
type Store struct {
	ds datastore.TxnDatastore
}

// New returns a new Store backed by `ds`.
func New(ds datastore.TxnDatastore) (*Store, error) {
	s := &Store{
		ds: ds,
	}
	return s, nil
}

// Save saves the provided BrokerRequest.
func (s *Store) Save(ctx context.Context, sr broker.BrokerRequest) error {
	buf, err := json.Marshal(sr)
	if err != nil {
		return fmt.Errorf("marshaling broker request: %s", err)
	}

	srKey := keyBrokerRequest(sr.ID)
	if err := s.ds.Put(srKey, buf); err != nil {
		return fmt.Errorf("put in datastore: %s", err)
	}

	return nil
}

// Get gets a BrokerRequest with the specified `id`. If not found returns ErrNotFound.
func (s *Store) Get(ctx context.Context, id broker.BrokerRequestID) (broker.BrokerRequest, error) {
	key := keyBrokerRequest(id)
	buf, err := s.ds.Get(key)
	if err == datastore.ErrNotFound {
		return broker.BrokerRequest{}, ErrNotFound
	}
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("get broker request from datstore: %s", err)
	}

	var sr broker.BrokerRequest
	if err := json.Unmarshal(buf, &sr); err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("unmarshaling broker request: %s", err)
	}

	return sr, nil
}

func keyBrokerRequest(ID broker.BrokerRequestID) datastore.Key {
	return prefixBrokerRequest.ChildString(string(ID))
}
