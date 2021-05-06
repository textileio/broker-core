package srstore

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
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
		br.Status = broker.RequestPreparing
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
	if err := saveStorageDeal(txn, sd); err != nil {
		return fmt.Errorf("saving storage deal: %s", err)
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

func (s *Store) StorageDealToAuctioning(
	ctx context.Context,
	id broker.StorageDealID,
	pieceCid cid.Cid,
	pieceSize uint64) error {
	sd, err := s.GetStorageDeal(ctx, id)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}

	// Take care of correct state transitions.
	switch sd.Status {
	case broker.StorageDealPreparing:
		// All good here, the status we would expect to transition from.

		// Validate anyway that the fields we expect to populate are free.
		// Panic mode here.
		if sd.PieceCid.Defined() || sd.PieceSize > 0 {
			return fmt.Errorf("piece cid and size should be empty: %s %d", sd.PieceCid, sd.PieceSize)
		}

		// Continue with happy path.
	case broker.StorageDealAuctioning:
		// Seems like we're trying to transition to the same status.
		// Most probably Piecer is doing a retry on notifying us, possibly because it didn't
		// receive our answer before.

		// Let's check that things are coherent, if not, error.
		if sd.PieceCid != pieceCid {
			return fmt.Errorf("piececid different from registered: %s %s", sd.pieceCid, pieceCid)
		}
		if sd.PieceSize != pieceSize {
			return fmt.Errorf("piece size different from registered: %s %s", sd.PieceSize, pieceSize)
		}

		// So the Piecer simply notified us with the same data. That should be fine, can be considered
		// a noop.

		return nil
	default:
		return fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
	}

	sd.PieceCid = pieceCid
	sd.PieceSize = pieceSize
	sd.UpdatedAt = time.Now()
	if err := s.saveStorageDeal(s.ds, sd); err != nil {
		return fmt.Errorf("save storage deal: %s", err)
	}

	return nil
}

func (s *Store) StorageDealToDealMaking(ctx context.Context, auction broker.Auction) error {
	sd, err := s.GetStorageDeal(ctx, auction.StorageDealID)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}

	// Take care of correct state transitions.
	switch sd.Status {
	case broker.StorageDealAuctioning:
		if sd.Auction.ID != "" {
			return fmt.Errorf("storage deal auction data isn't empty: %s", sd.Auction.ID)
		}

		// Continue with happy path.
	case broker.StorageDealDealMaking:
		// Seems like we're trying to transition to the same status.
		// Most probably Piecer is doing a retry on notifying us, possibly because it didn't
		// receive our answer before.

		// Let's check that things are coherent, if not, error.
		if sd.Auction.ID != auction.ID {
			return fmt.Errorf("signaled of another winning auction: %s %s", auction.ID, sd.Auction.ID)
		}

		// So the Piecer simply notified us with the same data. That should be fine, can be considered
		// a noop.

		return nil
	default:
		return fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
	}

	sd.Auction = auction
	sd.UpdatedAt = time.Now()
	if err := s.saveStorageDeal(s.ds, sd); err != nil {
		return fmt.Errorf("save storage deal: %s", err)
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

func (s *Store) saveStorageDeal(w datastore.Write, sd broker.StorageDeal) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(sd); err != nil {
		return fmt.Errorf("encoding gob: %s", err)
	}
	if err := w.Put(keyStorageDeal(sd.ID), buf.Bytes()); err != nil {
		return fmt.Errorf("saving storage deal in datastore: %s", err)
	}
	return nil
}

func keyBrokerRequest(ID broker.BrokerRequestID) datastore.Key {
	return prefixBrokerRequest.ChildString(string(ID))
}

func keyStorageDeal(ID broker.StorageDealID) datastore.Key {
	return prefixBrokerRequest.ChildString(string(ID))
}
