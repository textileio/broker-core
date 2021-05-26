package store

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	logger "github.com/ipfs/go-log/v2"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/dshelper/txndswrap"
)

type (
	status int

	// UnpreparedStorageDealID is an identifier for a pending storage deal
	// to be prepared.
	UnpreparedStorageDealID string
)

const (
	statusPending status = iota
	statusInProgress
)

// UnpreparedStorageDeal contains a StorageDeal data to be
// prepared for Filecoin onboarding.
type UnpreparedStorageDeal struct {
	ID            UnpreparedStorageDealID
	Status        status
	StorageDealID broker.StorageDealID
	DataCid       cid.Cid
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

var (
	log = logger.Logger("piecer/store")

	dsPrefixPendingUSD    = datastore.NewKey("/usd/pending")
	dsPrefixInProgressUSD = datastore.NewKey("/usd/inprogress")
)

// Store persists unprepared storage deals.
type Store struct {
	ds txndswrap.TxnDatastore

	lock    sync.Mutex
	entropy *ulid.MonotonicEntropy
}

// New returns a new Store.
func New(ds txndswrap.TxnDatastore) *Store {
	return &Store{
		ds: ds,
	}
}

// Create creates a new pending data to be prepared.
func (s *Store) Create(sdID broker.StorageDealID, dataCid cid.Cid) error {
	newID, err := s.newID()
	if err != nil {
		return fmt.Errorf("generating new id: %s", err)
	}

	now := time.Now()
	usd := UnpreparedStorageDeal{
		ID:            UnpreparedStorageDealID(newID),
		Status:        statusPending,
		StorageDealID: sdID,
		DataCid:       dataCid,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %s", err)
	}
	defer txn.Discard()

	key := makePendingUSDKey(usd.ID)
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(usd); err != nil {
		return fmt.Errorf("gob encoding: %s", err)
	}
	if err := txn.Put(key, buf.Bytes()); err != nil {
		return fmt.Errorf("put in datastore: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %s", err)
	}

	return nil
}

// GetNext returns the next pending unprepared storage deal. If no pending ones exist,
// it returns false as the second parameter.
func (s *Store) GetNext() (UnpreparedStorageDeal, bool, error) {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return UnpreparedStorageDeal{}, false, fmt.Errorf("creating txn: %s", err)
	}
	defer txn.Discard()

	q := query.Query{
		Prefix: dsPrefixPendingUSD.String(),
		Limit:  1,
	}
	res, err := txn.Query(q)
	if err != nil {
		return UnpreparedStorageDeal{}, false, fmt.Errorf("executing query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing query: %s", err)
		}
	}()

	item, ok := res.NextSync()
	if !ok {
		return UnpreparedStorageDeal{}, false, nil
	}

	var usd UnpreparedStorageDeal
	d := gob.NewDecoder(bytes.NewReader(item.Value))
	if err := d.Decode(&usd); err != nil {
		return UnpreparedStorageDeal{}, false, fmt.Errorf("unmarshaling gob: %s", err)
	}

	pendingKey := makePendingUSDKey(usd.ID)
	if err := txn.Delete(pendingKey); err != nil {
		return UnpreparedStorageDeal{}, false, fmt.Errorf("removing from pending namespace: %s", err)
	}

	usd.UpdatedAt = time.Now()
	usd.Status = statusInProgress
	inprogressKey := makeInProgressUSDKey(usd.ID)
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(usd); err != nil {
		return UnpreparedStorageDeal{}, false, fmt.Errorf("gob encoding: %s", err)
	}
	if err := txn.Put(inprogressKey, buf.Bytes()); err != nil {
		return UnpreparedStorageDeal{}, false, fmt.Errorf("put in datastore: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return UnpreparedStorageDeal{}, false, fmt.Errorf("committing txn: %s", err)
	}

	return usd, true, nil
}

// Delete removes an in-progress unprepared storage deal.
func (s *Store) Delete(id UnpreparedStorageDealID) error {
	inprogressKey := makeInProgressUSDKey(id)
	if err := s.ds.Delete(inprogressKey); err != nil {
		return fmt.Errorf("deleting in-progress unprepared storage deal: %s", err)
	}
	return nil
}

// MoveToPending moves an unprepared storage deal that's currently in-progress to
// pending.
func (s *Store) MoveToPending(id UnpreparedStorageDealID) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %s", err)
	}
	defer txn.Discard()

	inprogressKey := makeInProgressUSDKey(id)
	buf, err := txn.Get(inprogressKey)
	if err != nil {
		return fmt.Errorf("get in-progress unprepared storage deal: %s", err)
	}
	var usd UnpreparedStorageDeal
	d := gob.NewDecoder(bytes.NewReader(buf))
	if err := d.Decode(&usd); err != nil {
		return fmt.Errorf("unmarshaling gob: %s", err)
	}

	if err := txn.Delete(inprogressKey); err != nil {
		return fmt.Errorf("removing from in-progress namespace: %s", err)
	}

	usd.Status = statusPending
	usd.UpdatedAt = time.Now()

	pendingKey := makePendingUSDKey(usd.ID)
	var buf2 bytes.Buffer
	if err := gob.NewEncoder(&buf2).Encode(usd); err != nil {
		return fmt.Errorf("gob encoding: %s", err)
	}
	if err := txn.Put(pendingKey, buf2.Bytes()); err != nil {
		return fmt.Errorf("put in datastore: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %s", err)
	}

	return nil
}

func makePendingUSDKey(id UnpreparedStorageDealID) datastore.Key {
	return dsPrefixPendingUSD.ChildString(string(id))
}

func makeInProgressUSDKey(id UnpreparedStorageDealID) datastore.Key {
	return dsPrefixInProgressUSD.ChildString(string(id))
}

func (s *Store) newID() (string, error) {
	s.lock.Lock()

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
