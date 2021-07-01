package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/gob"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
	logger "github.com/textileio/go-log/v2"
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
	// Namespace "/storage-deal/{id}" contains
	// the current `BrokerRequest` data for an `id`.
	prefixStorageDeal = datastore.NewKey("storage-deal")

	log = logger.Logger("store")
)

// Store provides a persistent layer for broker requests.
type Store struct {
	ds datastore.TxnDatastore

	lock    sync.Mutex
	entropy *ulid.MonotonicEntropy
}

// New returns a new Store backed by `ds`.
func New(ds datastore.TxnDatastore) (*Store, error) {
	s := &Store{
		ds: ds,
	}
	return s, nil
}

// SaveBrokerRequest saves the provided BrokerRequest.
func (s *Store) SaveBrokerRequest(_ context.Context, br broker.BrokerRequest) error {
	ibr := castToInternalBrokerRequest(br)
	return saveBrokerRequest(s.ds, ibr)
}

// GetBrokerRequest gets a BrokerRequest with the specified `id`. If not found returns ErrNotFound.
func (s *Store) GetBrokerRequest(ctx context.Context, id broker.BrokerRequestID) (broker.BrokerRequest, error) {
	ibr, err := getBrokerRequest(s.ds, id)
	if err != nil {
		return broker.BrokerRequest{}, err
	}
	return castToBrokerRequest(ibr), nil
}

// CreateStorageDeal persists a storage deal. It populates the sd.ID field with the corresponding id.
func (s *Store) CreateStorageDeal(ctx context.Context, sd *broker.StorageDeal) error {
	if sd.ID != "" {
		return fmt.Errorf("storage deal id must be empty")
	}
	newID, err := s.newID()
	if err != nil {
		return fmt.Errorf("generating id: %s", err)
	}
	sd.ID = auction.StorageDealID(newID)

	start := time.Now()
	defer log.Debugf(
		"creating storage deal %s with group size %d took %dms",
		sd.ID, len(sd.BrokerRequestIDs),
		time.Since(start).Milliseconds(),
	)

	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating datastore transaction: %s", err)
	}
	defer txn.Discard()

	// 1- Get all involved BrokerRequests and validate that their in the correct
	// statuses, and nothing unexpected/invalid is going on.
	brs := make([]brokerRequest, len(sd.BrokerRequestIDs))
	now := time.Now()
	for i, brID := range sd.BrokerRequestIDs {
		br, err := getBrokerRequest(txn, brID)
		if err == ErrNotFound {
			return fmt.Errorf("unknown broker request id %s: %w", brID, ErrStorageDealContainsUnknownBrokerRequest)
		}

		// 2- Link all BrokerRequests with the StorageDeal, and change their status
		// to `Preparing`, since they should mirror the StorageDeal status.
		switch sd.Status {
		case broker.StorageDealPreparing:
			br.Status = broker.RequestPreparing
		case broker.StorageDealAuctioning:
			br.Status = broker.RequestAuctioning
		default:
			return fmt.Errorf("unexpected storage deal initial status %d", sd.Status)
		}
		br.UpdatedAt = now
		br.StorageDealID = sd.ID

		brs[i] = br
	}

	// 3- Persist the StorageDeal.
	dsources := sources{}
	if sd.Sources.CARURL != nil {
		url := sd.Sources.CARURL.URL.String()
		dsources.CARURL = &url
	}
	if sd.Sources.CARIPFS != nil {
		dsources.CARIPFS = &carIPFS{
			Cid:        sd.Sources.CARIPFS.Cid.String(),
			Multiaddrs: make([]string, len(sd.Sources.CARIPFS.Multiaddrs)),
		}
		for i, maddr := range sd.Sources.CARIPFS.Multiaddrs {
			dsources.CARIPFS.Multiaddrs[i] = maddr.String()
		}
	}
	isd := storageDeal{
		ID:                 sd.ID,
		Status:             sd.Status,
		BrokerRequestIDs:   make([]broker.BrokerRequestID, len(brs)),
		RepFactor:          sd.RepFactor,
		DealDuration:       sd.DealDuration,
		PayloadCid:         sd.PayloadCid,
		PieceCid:           sd.PieceCid,
		PieceSize:          sd.PieceSize,
		DisallowRebatching: sd.DisallowRebatching,
		AuctionRetries:     sd.AuctionRetries,
		FilEpochDeadline:   sd.FilEpochDeadline,
		Sources:            dsources,
		CreatedAt:          sd.CreatedAt,
		UpdatedAt:          sd.UpdatedAt,
	}
	copy(isd.BrokerRequestIDs, sd.BrokerRequestIDs)
	if err := saveStorageDeal(txn, isd); err != nil {
		return fmt.Errorf("saving storage deal: %s", err)
	}

	for _, br := range brs {
		if err := saveBrokerRequest(txn, br); err != nil {
			return fmt.Errorf("saving broker request: %s", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// StorageDealToAuctioning moves a storage deal and the underlying broker requests
// to Auctioning status.
func (s *Store) StorageDealToAuctioning(
	ctx context.Context,
	id auction.StorageDealID,
	pieceCid cid.Cid,
	pieceSize uint64) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating transaction: %s", err)
	}
	defer txn.Discard()

	sd, err := getStorageDeal(txn, id)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}

	// Take care of correct state transitions.
	switch sd.Status {
	case broker.StorageDealPreparing:
		if sd.PieceCid.Defined() || sd.PieceSize > 0 {
			return fmt.Errorf("piece cid and size should be empty: %s %d", sd.PieceCid, sd.PieceSize)
		}
	case broker.StorageDealAuctioning:
		if sd.PieceCid != pieceCid {
			return fmt.Errorf("piececid different from registered: %s %s", sd.PieceCid, pieceCid)
		}
		if sd.PieceSize != pieceSize {
			return fmt.Errorf("piece size different from registered: %d %d", sd.PieceSize, pieceSize)
		}
		return nil
	default:
		return fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
	}

	now := time.Now()
	sd.Status = broker.StorageDealAuctioning
	sd.PieceCid = pieceCid
	sd.PieceSize = pieceSize
	sd.UpdatedAt = now

	if err := saveStorageDeal(txn, sd); err != nil {
		return fmt.Errorf("save storage deal: %s", err)
	}

	for _, brID := range sd.BrokerRequestIDs {
		br, err := getBrokerRequest(txn, brID)
		if err != nil {
			return fmt.Errorf("getting broker request: %s", err)
		}
		br.Status = broker.RequestAuctioning
		br.UpdatedAt = now
		if err := saveBrokerRequest(txn, br); err != nil {
			return fmt.Errorf("saving broker request: %s", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// StorageDealError moves a storage deal to an error status with a specified error cause.
// The underlying broker requests are moved to Batching status. The caller is responsible to
// schedule again this broker requests to Packer.
func (s *Store) StorageDealError(
	ctx context.Context,
	id auction.StorageDealID,
	errorCause string,
	rebatch bool) ([]broker.BrokerRequestID, error) {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return nil, fmt.Errorf("creating transaction: %s", err)
	}
	defer txn.Discard()

	sd, err := getStorageDeal(txn, id)
	if err != nil {
		return nil, fmt.Errorf("get storage deal: %s", err)
	}

	// 1. Verify some pre-state conditions.
	switch sd.Status {
	case broker.StorageDealAuctioning, broker.StorageDealDealMaking:
		if sd.Error != "" {
			return nil, fmt.Errorf("error cause should be empty: %s", sd.Error)
		}
	case broker.StorageDealError:
		if sd.Error != errorCause {
			return nil, fmt.Errorf("the error cause is different from the registeredon : %s %s", sd.Error, errorCause)
		}
		return sd.BrokerRequestIDs, nil
	default:
		return nil, fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
	}

	// 2. Move the StorageDeal to StorageDealError with the error cause.
	now := time.Now()
	sd.Status = broker.StorageDealError
	sd.Error = errorCause
	sd.UpdatedAt = now

	if err := saveStorageDeal(txn, sd); err != nil {
		return nil, fmt.Errorf("save storage deal: %s", err)
	}

	// 3. Move every underlying BrokerRequest to batching again, since they will be re-batched.
	for _, brID := range sd.BrokerRequestIDs {
		br, err := getBrokerRequest(txn, brID)
		if err != nil {
			return nil, fmt.Errorf("getting broker request: %s", err)
		}
		br.Status = broker.RequestError
		if rebatch {
			br.Status = broker.RequestBatching
			br.RebatchCount++
			br.ErrCause = errorCause
		}
		br.UpdatedAt = now
		if err := saveBrokerRequest(txn, br); err != nil {
			return nil, fmt.Errorf("saving broker request: %s", err)
		}
	}

	// 4. Mark the batchCid as unpinnable, since it won't be used anymore for auctions or deals.
	unpinID, err := s.newID()
	if err != nil {
		return nil, fmt.Errorf("generating id for unpin job: %s", err)
	}
	unpin := UnpinJob{
		ID:        UnpinJobID(unpinID),
		Cid:       sd.PayloadCid,
		Type:      UnpinTypeBatch,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := saveUnpinJob(txn, unpin); err != nil {
		return nil, fmt.Errorf("saving unpin job: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %s", err)
	}

	return sd.BrokerRequestIDs, nil
}

// StorageDealSuccess moves a storage deal and the underlying broker requests to
// Success status.
func (s *Store) StorageDealSuccess(ctx context.Context, id auction.StorageDealID) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating transaction: %s", err)
	}
	defer txn.Discard()

	sd, err := getStorageDeal(txn, id)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}

	// Take care of correct state transitions.
	switch sd.Status {
	case broker.StorageDealDealMaking:
		if sd.Error != "" {
			return fmt.Errorf("error cause should be empty: %s", sd.Error)
		}
	case broker.StorageDealSuccess:
		return nil
	default:
		return fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
	}

	now := time.Now()
	sd.Status = broker.StorageDealSuccess
	sd.UpdatedAt = now

	if err := saveStorageDeal(txn, sd); err != nil {
		return fmt.Errorf("save storage deal: %s", err)
	}

	for _, brID := range sd.BrokerRequestIDs {
		br, err := getBrokerRequest(txn, brID)
		if err != nil {
			return fmt.Errorf("getting broker request: %s", err)
		}
		br.Status = broker.RequestSuccess
		br.UpdatedAt = now
		if err := saveBrokerRequest(txn, br); err != nil {
			return fmt.Errorf("saving broker request: %s", err)
		}
		unpinID, err := s.newID()
		if err != nil {
			return fmt.Errorf("generating id for unpin job: %s", err)
		}
		unpin := UnpinJob{
			ID:        UnpinJobID(unpinID),
			Cid:       br.DataCid,
			Type:      UnpinTypeData,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := saveUnpinJob(txn, unpin); err != nil {
			return fmt.Errorf("saving unpin job: %s", err)
		}
	}

	unpinID, err := s.newID()
	if err != nil {
		return fmt.Errorf("generating id for unpin job: %s", err)
	}
	unpin := UnpinJob{
		ID:        UnpinJobID(unpinID),
		Cid:       sd.PayloadCid,
		Type:      UnpinTypeBatch,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := saveUnpinJob(txn, unpin); err != nil {
		return fmt.Errorf("saving unpin job: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// CountAuctionRetry increases the number of auction retries counter for a storage deal.
func (s *Store) CountAuctionRetry(ctx context.Context, sdID auction.StorageDealID) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating transaction: %s", err)
	}
	defer txn.Discard()

	sd, err := getStorageDeal(txn, sdID)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}

	sd.AuctionRetries++
	sd.UpdatedAt = time.Now()

	if err := saveStorageDeal(txn, sd); err != nil {
		return fmt.Errorf("save storage deal: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// AddMinerDeals includes new deals from a finalized auction.
func (s *Store) AddMinerDeals(ctx context.Context, auction broker.ClosedAuction) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating transaction: %s", err)
	}
	defer txn.Discard()

	sd, err := getStorageDeal(txn, auction.StorageDealID)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}

	// Take care of correct state transitions.
	if sd.Status != broker.StorageDealAuctioning && sd.Status != broker.StorageDealDealMaking {
		return fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
	}

	now := time.Now()
	// Add winning bids to list of deals.
	for bidID, bid := range auction.WinningBids {
		sd.Deals = append(sd.Deals,
			minerDeal{
				StorageDealID: auction.StorageDealID,
				AuctionID:     auction.ID,
				BidID:         bidID,
				CreatedAt:     now,
				UpdatedAt:     now,
				Miner:         bid.MinerAddr,
			})
	}

	sd.UpdatedAt = now

	moveBrokerRequestsToDealMaking := false
	if sd.Status == broker.StorageDealAuctioning {
		sd.Status = broker.StorageDealDealMaking
		moveBrokerRequestsToDealMaking = true
	}

	if err := saveStorageDeal(txn, sd); err != nil {
		return fmt.Errorf("save storage deal: %s", err)
	}

	// If the storage-deal was already in StorageDealDealMaking, then we're
	// just adding more winning bids from new auctions we created to statisfy the
	// replication factor. That case means we already updated the underlying
	// broker-requests status.
	// This conditional is to only move the broker-request on the first successful auction
	// that might happen. On further auctions, we don't need to do this again.
	if moveBrokerRequestsToDealMaking {
		for _, brID := range sd.BrokerRequestIDs {
			br, err := getBrokerRequest(txn, brID)
			if err != nil {
				return fmt.Errorf("getting broker request: %s", err)
			}
			br.Status = broker.RequestDealMaking
			br.UpdatedAt = now
			if err := saveBrokerRequest(txn, br); err != nil {
				return fmt.Errorf("saving broker request: %s", err)
			}
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// GetStorageDeal gets an existing storage deal by id. If the storage deal doesn't exists, it returns
// ErrNotFound.
func (s *Store) GetStorageDeal(ctx context.Context, id auction.StorageDealID) (broker.StorageDeal, error) {
	isd, err := getStorageDeal(s.ds, id)
	if err != nil {
		return broker.StorageDeal{}, fmt.Errorf("get storage-deal from datastore: %s", err)
	}

	return castToStorageDeal(isd)
}

// SaveFinalizedDeal saves a new finalized (succeeded or errored) auction deal
// into the storage deal.
func (s *Store) SaveFinalizedDeal(fad broker.FinalizedAuctionDeal) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating transaction: %s", err)
	}
	defer txn.Discard()

	sd, err := getStorageDeal(txn, fad.StorageDealID)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}

	idx := -1
	for i := range sd.Deals {
		if sd.Deals[i].Miner == fad.Miner {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("deal not found: %v", fad)
	}
	sd.Deals[idx].DealExpiration = fad.DealExpiration
	sd.Deals[idx].DealID = fad.DealID
	sd.Deals[idx].ErrorCause = fad.ErrorCause

	if err := saveStorageDeal(txn, sd); err != nil {
		return fmt.Errorf("saving storage deal in datastore: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

func getBrokerRequest(r datastore.Read, id broker.BrokerRequestID) (brokerRequest, error) {
	key := keyBrokerRequest(id)
	buf, err := r.Get(key)
	if err == datastore.ErrNotFound {
		return brokerRequest{}, ErrNotFound
	}
	if err != nil {
		return brokerRequest{}, fmt.Errorf("get broker request from datstore: %s", err)
	}

	var sr brokerRequest
	dec := gob.NewDecoder(bytes.NewReader(buf))
	if err := dec.Decode(&sr); err != nil {
		return brokerRequest{}, fmt.Errorf("gob decoding: %s", err)
	}

	return sr, nil
}

func saveBrokerRequest(w datastore.Write, br brokerRequest) error {
	if err := br.validate(); err != nil {
		return fmt.Errorf("broker request is invalid: %s", err)
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(br); err != nil {
		return fmt.Errorf("encoding gob: %s", err)
	}
	srKey := keyBrokerRequest(br.ID)
	if err := w.Put(srKey, buf.Bytes()); err != nil {
		return fmt.Errorf("put in datastore: %s", err)
	}

	return nil
}

func saveStorageDeal(w datastore.Write, sd storageDeal) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(sd); err != nil {
		return fmt.Errorf("encoding gob: %s", err)
	}
	if err := w.Put(keyStorageDeal(sd.ID), buf.Bytes()); err != nil {
		return fmt.Errorf("saving storage deal in datastore: %s", err)
	}

	return nil
}

func saveUnpinJob(w datastore.Write, uj UnpinJob) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(uj); err != nil {
		return fmt.Errorf("encoding gob: %s", err)
	}
	if err := w.Put(keyPendingUnpinJob(uj.ID), buf.Bytes()); err != nil {
		return fmt.Errorf("saving storage deal in datastore: %s", err)
	}

	return nil
}

func getStorageDeal(r datastore.Read, id auction.StorageDealID) (storageDeal, error) {
	var sd storageDeal
	buf, err := r.Get(keyStorageDeal(id))
	if err == datastore.ErrNotFound {
		return storageDeal{}, ErrNotFound
	}
	dec := gob.NewDecoder(bytes.NewReader(buf))
	if err := dec.Decode(&sd); err != nil {
		return storageDeal{}, fmt.Errorf("unmarshaling storage deal: %s", err)
	}

	return sd, nil
}

func keyBrokerRequest(ID broker.BrokerRequestID) datastore.Key {
	return prefixBrokerRequest.ChildString(string(ID))
}

func keyStorageDeal(ID auction.StorageDealID) datastore.Key {
	return prefixStorageDeal.ChildString(string(ID))
}
func (s *Store) newID() (string, error) {
	s.lock.Lock()
	// Not deferring unlock since can be recursive.

	if s.entropy == nil {
		s.entropy = ulid.Monotonic(rand.Reader, 0)
	}
	id, err := ulid.New(ulid.Timestamp(time.Now().UTC()), s.entropy)
	if errors.Is(err, ulid.ErrMonotonicOverflow) {
		s.entropy = nil
		s.lock.Unlock()
		return s.newID()
	} else if err != nil {
		s.lock.Unlock()
		return "", fmt.Errorf("generating id: %v", err)
	}
	s.lock.Unlock()
	return strings.ToLower(id.String()), nil
}
