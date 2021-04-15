package srstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ipfs/go-datastore"
	logger "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/broker"
)

var (
	// ErrNotFound is returned if the broker request doesn't exist.
	ErrNotFound = fmt.Errorf("broker request not found")
	// ErrStorageDealContainsUnknownBrokerRequest is returned if a storage deal contains an
	// unknown broker request.
	ErrStorageDealContainsUnknownBrokerRequest = fmt.Errorf("storage deal contains an unknown broker request")

	// Namespace "/broker-request/{id}" contains
	// the current `BrokerRequest` data for an `id`.
	prefixBrokerRequest = datastore.NewKey("broker-request")

	log = logger.Logger("srstore")
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

// SaveBrokerRequest saves the provided BrokerRequest.
func (s *Store) SaveBrokerRequest(ctx context.Context, br broker.BrokerRequest) error {
	buf, err := json.Marshal(br)
	if err != nil {
		return fmt.Errorf("marshaling broker request: %s", err)
	}

	srKey := keyBrokerRequest(br.ID)
	if err := s.ds.Put(srKey, buf); err != nil {
		return fmt.Errorf("put in datastore: %s", err)
	}

	return nil
}

// GetBrokerRequest gets a BrokerRequest with the specified `id`. If not found returns ErrNotFound.
func (s *Store) GetBrokerRequest(ctx context.Context, id broker.BrokerRequestID) (broker.BrokerRequest, error) {
	return s.getBrokerRequest(ctx, id)
}

// CreateStorageDeal persists a storage deal.
func (s *Store) CreateStorageDeal(ctx context.Context, sd broker.StorageDeal) error {
	start := time.Now()
	defer log.Debugf(
		"creating storage deal %s with group size %d took %dms",
		sd.ID, len(sd.BrokerRequestIDs),
		time.Since(start).Milliseconds(),
	)
	now := time.Now()

	// 1- Get all involved BrokerRequests and validate that their in the correct
	// statuses, and nothing unexpected/invalid is going on.
	brs := make([]broker.BrokerRequest, len(sd.BrokerRequestIDs))
	for i, brID := range sd.BrokerRequestIDs {
		br, err := s.getBrokerRequest(ctx, brID)
		if err == ErrNotFound {
			return fmt.Errorf("unknown broker request id %s: %w", brID, ErrStorageDealContainsUnknownBrokerRequest)
		}

		// 2- Link all BrokerRequests with the StorageDeal, and change their status
		// to `Preparing`, since they should mirror the StorageDeal status.
		br.Status = broker.BrokerRequestPreparing
		br.UpdatedAt = now
		br.StorageDealID = sd.ID

		brs[i] = br
	}

	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating datastore transaction: %s", err)
	}
	defer txn.Discard()

	// 3- Persist the StorageDeal.
	buf, err := json.Marshal(sd)
	if err != nil {
		return fmt.Errorf("marshaling storage deal: %s", err)
	}
	if err := txn.Put(keyStorageDeal(sd.ID), buf); err != nil {
		return fmt.Errorf("saving storage deal in datastore: %s", err)
	}

	for _, br := range brs {
		buf, err := json.Marshal(br)
		if err != nil {
			return fmt.Errorf("marshaling broker request: %s", err)
		}

		srKey := keyBrokerRequest(br.ID)
		if err := txn.Put(srKey, buf); err != nil {
			return fmt.Errorf("put in datastore: %s", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// GetStorageDeal gets an existing storage deal by id. If the storage deal doesn't exists, it returns
// ErrNotFound.
func (s *Store) GetStorageDeal(ctx context.Context, id broker.StorageDealID) (broker.StorageDeal, error) {
	var sd broker.StorageDeal
	buf, err := s.ds.Get(keyStorageDeal(id))
	if err == datastore.ErrNotFound {
		return broker.StorageDeal{}, ErrNotFound
	}
	if err := json.Unmarshal(buf, &sd); err != nil {
		return broker.StorageDeal{}, fmt.Errorf("unmarshaling storage deal: %s", err)
	}

	return sd, nil
}

func (s *Store) getBrokerRequest(_ context.Context, id broker.BrokerRequestID) (broker.BrokerRequest, error) {
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

func keyStorageDeal(ID broker.StorageDealID) datastore.Key {
	return prefixBrokerRequest.ChildString(string(ID))
}
