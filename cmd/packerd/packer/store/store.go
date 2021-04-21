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
	"github.com/oklog/ulid/v2"
	"github.com/textileio/broker-core/broker"
)

var (
	// ErrNotFound is returned when dequeuing an element that doesn't exist.
	ErrNotFound = fmt.Errorf("batchable broker request doesn't exist")

	dsPrefixBatchableBrokerRequest = datastore.NewKey("/bbr")
)

// BatchableBrokerRequest is a broker-request that's pending to be
// batched.
type BatchableBrokerRequest struct {
	ID              string
	BrokerRequestID broker.BrokerRequestID
	DataCid         cid.Cid
}

type Store struct {
	ds datastore.TxnDatastore

	// We keep an in-memory representation of the data.
	// BatchableBrokerRequest is a pretty minimal datastructure.
	// Even if we have millions of rows, that would be a small amount of memory usage.
	// Allow us to not hit hard the datastore, and be flexible with limited query
	// capabilities of go-datastore for the future too.
	lock    sync.Mutex
	queue   []BatchableBrokerRequest
	entropy *ulid.MonotonicEntropy
}

func New(ds datastore.TxnDatastore) (*Store, error) {
	s := &Store{
		ds: ds,
	}
	if err := s.loadCache(); err != nil {
		return nil, fmt.Errorf("loading in-memory cache: %s", err)
	}

	return s, nil
}

// Enqueue enqueue a new batchable broker requests in persistent store.
func (s *Store) Enqueue(br BatchableBrokerRequest) error {
	newID, err := s.newID()
	if err != nil {
		return fmt.Errorf("generating new id: %s", err)
	}
	br.ID = newID
	if err := s.save(br); err != nil {
		return fmt.Errorf("saving batchable broker request in datastore: %s", err)
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

// Dequeue removes the provided BatchableBrokerRequests.
func (s *Store) Dequeue(bbrs []BatchableBrokerRequest) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating transaction: %s", err)
	}
	defer txn.Discard()

	for _, bbr := range bbrs {
		key := makeBatchableBrokerRequestKey(bbr.ID)
		exists, err := txn.Has(key)
		if err != nil {
			return fmt.Errorf("checking if batchable broker request exists: %s", err)
		}
		if !exists {
			return ErrNotFound
		}
		if err := txn.Delete(key); err != nil {
			return fmt.Errorf("deleting batchable broker request in txn: %s", err)
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
	// We removed without keeping order, sort again.
	// This sounds like a good strategy. We could also remove while keeping order,
	// but that sounds more expensive since we're copying subslices around.
	// Easy to switch if we discover otherwise.
	sort.Slice(s.queue, func(i, j int) bool {
		return s.queue[i].ID < s.queue[j].ID
	})
	s.lock.Unlock()

	return nil
}

// Iterator provides an iterator to walk throught available
// BatchableStorageRequest in ascending order.
type Iterator struct {
	s       *Store
	nextIdx int
}

// NewIterator returns a new iterator.
func (s *Store) NewIterator() *Iterator {
	return &Iterator{s: s}
}

// Next returns the next available BatchableBrokerRequest. If the end of the list
// is found, then false is retured.
func (i *Iterator) Next() (BatchableBrokerRequest, bool) {
	i.s.lock.Lock()
	defer i.s.lock.Unlock()
	if i.nextIdx == len(i.s.queue) {
		return BatchableBrokerRequest{}, false
	}
	brr := i.s.queue[i.nextIdx]
	i.nextIdx++

	return brr, true
}

func (s *Store) save(br BatchableBrokerRequest) error {
	key := makeBatchableBrokerRequestKey(br.ID)
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
	q := query.Query{Prefix: dsPrefixBatchableBrokerRequest.String()}
	res, err := s.ds.Query(q)
	if err != nil {
		return fmt.Errorf("creating query: %s", err)
	}
	defer res.Close()

	for item := range res.Next() {
		if item.Error != nil {
			return fmt.Errorf("fetching query item result: %s", item.Error)
		}
		var bbr BatchableBrokerRequest
		d := gob.NewDecoder(bytes.NewReader(item.Value))
		if err := d.Decode(&bbr); err != nil {
			return fmt.Errorf("unmarshaling gob: %s", err)
		}
		s.queue = append(s.queue, bbr)
	}

	return nil
}

func makeBatchableBrokerRequestKey(id string) datastore.Key {
	return dsPrefixBatchableBrokerRequest.ChildString(id)
}
