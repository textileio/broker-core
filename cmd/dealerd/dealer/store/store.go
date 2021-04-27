package store

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	logger "github.com/ipfs/go-log/v2"
	"github.com/oklog/ulid/v2"
)

var (
	ErrNotFound = fmt.Errorf("bid doesn't exist")

	dsPrefixBid = datastore.NewKey("/bid")

	log = logger.Logger("dealer/store")
)

type Bid struct {
	ID string

	DataCid           cid.Cid
	DealStartEpoch    int64
	DealCommP         cid.Cid
	DealSize          int64
	DealPricePerEpoch int64
	DealDuration      int64
	Verified          bool
	Miner             string
}

// Store provides persistent storage for Bids.
type Store struct {
	ds datastore.TxnDatastore

	lock    sync.Mutex
	queue   []Bid
	entropy *ulid.MonotonicEntropy
}

// New returns a *Store.
func New(ds datastore.TxnDatastore) (*Store, error) {
	s := &Store{
		ds: ds,
	}
	if err := s.loadCache(); err != nil {
		return nil, fmt.Errorf("loading in-memory cache: %s", err)
	}

	return s, nil
}

// Enqueue enqueue a new bid ready for execution.
func (s *Store) Enqueue(br Bid) error {
	newID, err := s.newID()
	if err != nil {
		return fmt.Errorf("generating new id: %s", err)
	}
	br.ID = newID
	if err := s.save(br); err != nil {
		return fmt.Errorf("saving bid in datastore: %s", err)
	}

	s.lock.Lock()
	s.queue = append(s.queue, br)
	// Since we generate the monotonic ID outside of the lock
	// there's a chance that appending to the `queue` might be out of
	// order if other `Enqueue` call is too fast.
	sort.Slice(s.queue, func(i, j int) bool {
		return s.queue[i].ID < s.queue[j].ID
	})
	s.lock.Unlock()

	return nil
}

// Dequeue removes the provided bids.
func (s *Store) Dequeue(bbrs []Bid) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating transaction: %s", err)
	}
	defer txn.Discard()

	for _, bbr := range bbrs {
		key := makeBidKey(bbr.ID)
		exists, err := txn.Has(key)
		if err != nil {
			return fmt.Errorf("checking if bid exists: %s", err)
		}
		if !exists {
			return ErrNotFound
		}
		if err := txn.Delete(key); err != nil {
			return fmt.Errorf("deleting bid in txn: %s", err)
		}
	}
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	s.lock.Lock()
	for _, bbr := range bbrs {
		for i := range s.queue {
			if s.queue[i].ID == bbr.ID {
				s.queue[i] = s.queue[len(s.queue)-1]
				s.queue = s.queue[:len(s.queue)-1]
				break
			}
		}
	}
	sort.Slice(s.queue, func(i, j int) bool {
		return s.queue[i].ID < s.queue[j].ID
	})
	s.lock.Unlock()

	return nil
}

func (s *Store) save(br Bid) error {
	key := makeBidKey(br.ID)
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(br); err != nil {
		return fmt.Errorf("gob encoding: %s", err)
	}
	if err := s.ds.Put(key, buf.Bytes()); err != nil {
		return fmt.Errorf("put in datastore: %s", err)
	}

	return nil
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

func (s *Store) loadCache() error {
	q := query.Query{
		Prefix: dsPrefixBid.String(),
		Orders: []query.Order{query.OrderByKey{}},
	}
	res, err := s.ds.Query(q)
	if err != nil {
		return fmt.Errorf("creating query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing query result: %s", err)
		}
	}()

	for item := range res.Next() {
		if item.Error != nil {
			return fmt.Errorf("fetching query item result: %s", item.Error)
		}
		var bbr Bid
		d := gob.NewDecoder(bytes.NewReader(item.Value))
		if err := d.Decode(&bbr); err != nil {
			return fmt.Errorf("unmarshaling gob: %s", err)
		}
		s.queue = append(s.queue, bbr)
	}

	return nil
}

func makeBidKey(id string) datastore.Key {
	return dsPrefixBid.ChildString(id)
}
