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
	dsq "github.com/ipfs/go-datastore/query"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/broker-core/broker"
	logger "github.com/textileio/go-log/v2"
)

var (
	// /batch/pending/<batch-id> (an open batch that will keep aggregating broker-requests until is closed).
	dsPrefixPendingBatch = datastore.NewKey("/batch/pending")
	// /batch/ready/<batch-id> (a batch ready to be prepared and signaled to the broker).
	dsPrefixReadyBatch = datastore.NewKey("/batch/ready")
	// /batch/executing/<batch-id> (a batch being processed).
	dsPrefixExecutingBatch = datastore.NewKey("/batch/executing")

	// /batch-br/<batch-id>/<batchable-broker-request-id> (a broker-request present in a batch).
	dsPrefixBatchBrokerRequest = datastore.NewKey("/batch-br")

	log = logger.Logger("packer/store")
)

type (
	// BatchID is an identifier for a Batch.
	BatchID                  string
	batchableBrokerRequestID string
)

// BatchableBrokerRequest is a broker-request that's pending to be
// batched.
type BatchableBrokerRequest struct {
	ID              batchableBrokerRequestID
	BrokerRequestID broker.BrokerRequestID
	DataCid         cid.Cid
	Size            int64
}

// Batch is a container for batchable broker requests.
type Batch struct {
	ID    BatchID
	Count int64
	Size  int64
}

// Store provides persistent storage for BatchableBrokerRequest.
type Store struct {
	ds                datastore.TxnDatastore
	batchMaxSize      int64
	batchReadyMinSize uint

	lock    sync.Mutex
	entropy *ulid.MonotonicEntropy
}

// New returns a *Store.
func New(ds datastore.TxnDatastore, batchMaxSize int64, batchReadyMinSize uint) (*Store, error) {
	if batchMaxSize < int64(batchReadyMinSize) {
		return nil, fmt.Errorf("batch max size can't be smaller than the minimum batch size")
	}
	s := &Store{
		ds:                ds,
		batchMaxSize:      batchMaxSize,
		batchReadyMinSize: batchReadyMinSize,
	}

	return s, nil
}

// Enqueue enqueue a new batchable broker requests in persistent store.
func (s *Store) Enqueue(br BatchableBrokerRequest) error {
	if br.Size > s.batchMaxSize {
		return fmt.Errorf("the data is bigger (%d) than the maximum allowed size (%d)", br.Size, s.batchMaxSize)
	}

	newID, err := s.newID()
	if err != nil {
		return fmt.Errorf("generating new id: %s", err)
	}
	br.ID = batchableBrokerRequestID(newID)

	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %s", err)
	}
	defer txn.Discard()

	// 1. Try getting an open batch where this data can fit-in.
	openBatch, ok, err := getOpenBatch(txn, br.Size, s.batchMaxSize)
	if err != nil {
		return fmt.Errorf("get or create open batch: %s", err)
	}

	// If we didn't find one, create a new open batch.
	if !ok {
		newID, err := s.newID()
		if err != nil {
			return fmt.Errorf("generating new id: %s", err)
		}
		openBatch = Batch{
			ID:   BatchID(newID),
			Size: 0,
		}

		key := makePendingBatchKey(openBatch.ID)
		if err := save(txn, key, openBatch); err != nil {
			return fmt.Errorf("put new batch in datastore: %s", err)
		}
		log.Debugf("created a new batch %s", openBatch.ID)
	}

	// 2. Update the open batch size.
	openBatch.Size += br.Size
	openBatch.Count++
	batchKey := makePendingBatchKey(openBatch.ID)

	// 3. If the batch reached the size threshold, close it.
	if uint(openBatch.Size) >= s.batchReadyMinSize {
		log.Debugf("batch has size %d > %d, moving to ready...", openBatch.Size, s.batchReadyMinSize)
		// 3.a Delete it from pending
		if err := txn.Delete(batchKey); err != nil {
			return fmt.Errorf("deleting pending batch: %s", err)
		}

		// 3.b Set the batchKey to be on the Ready namespace
		//     so the `save` below will save it in the new place.
		batchKey = makeReadyBatchKey(openBatch.ID)
	}

	// 4. Save the modified open (or ready!) batch to the datastore.
	if err := save(txn, batchKey, openBatch); err != nil {
		return fmt.Errorf("save batch with new size: %s", err)
	}

	// 5. Save the batchable broker request under that open batch-id.
	brKey := makeBatchableBrokerRequestKey(openBatch.ID, br.ID)
	if err := save(txn, brKey, br); err != nil {
		return fmt.Errorf("saving batchable broker request in datastore: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %s", err)
	}

	return nil
}

// GetNextReadyBatch returns the next batch that is ready to be packed. If none exists,
// it returns false in the third parameter.
func (s *Store) GetNextReadyBatch() (Batch, []BatchableBrokerRequest, bool, error) {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return Batch{}, nil, false, fmt.Errorf("creating transaction: %s", err)
	}
	defer txn.Discard()

	// 1. Try getting the first available ready batch.
	q := query.Query{
		Prefix: dsPrefixReadyBatch.String(),
		Orders: []dsq.Order{dsq.OrderByKey{}},
		Limit:  1,
	}
	res, err := txn.Query(q)
	if err != nil {
		return Batch{}, nil, false, fmt.Errorf("creating query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing get next ready batch query: %s", err)
		}
	}()

	var b Batch
	item, ok := res.NextSync()
	if item.Error != nil {
		return Batch{}, nil, false, fmt.Errorf("get item from query: %s", err)
	}

	// None available? Return with `false` and no error.
	if !ok {
		return Batch{}, nil, false, nil
	}

	// 2. We have one. Grab all the underlying batchable broker requests.
	dec := gob.NewDecoder(bytes.NewReader(item.Value))
	if err := dec.Decode(&b); err != nil {
		return Batch{}, nil, false, fmt.Errorf("decoding gob batch: %s", err)
	}

	bbrs := make([]BatchableBrokerRequest, 0, b.Count)
	q = query.Query{
		// We query for all the batchable broker request from the batch.
		// Recall: /batch-br/<batch-id>/<batchable-broker-request-id>
		Prefix: dsPrefixBatchBrokerRequest.ChildString(string(b.ID)).String(),
	}
	res, err = txn.Query(q)
	if err != nil {
		return Batch{}, nil, false, fmt.Errorf("execute bbrs for batch query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing bbrs for batch query: %s", err)
		}
	}()

	for item := range res.Next() {
		if item.Error != nil {
			return Batch{}, nil, false, fmt.Errorf("get item result from batch query: %s", item.Error)
		}
		var bbr BatchableBrokerRequest
		dec := gob.NewDecoder(bytes.NewReader(item.Value))
		if err := dec.Decode(&bbr); err != nil {
			return Batch{}, nil, false, fmt.Errorf("decoding gob batch batchable broker request: %s", err)
		}
		bbrs = append(bbrs, bbr)
	}

	// 3. Remove from Pending status.
	readyBatchKey := makeReadyBatchKey(b.ID)
	if err := txn.Delete(readyBatchKey); err != nil {
		return Batch{}, nil, false, fmt.Errorf("removing batch from ready state: %s", err)
	}

	// 4. Move to Executing status.
	executingBatchKey := makeExecutingBatchKey(b.ID)
	if err := save(txn, executingBatchKey, b); err != nil {
		return Batch{}, nil, false, fmt.Errorf("save batch moved to executing: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return Batch{}, nil, false, fmt.Errorf("committing txn: %s", err)
	}

	return b, bbrs, true, nil
}

// DeleteBatch delete a batch and all linked batchable broker requests from the store.
func (s *Store) DeleteBatch(bID BatchID) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %s", err)
	}

	executingKey := makeExecutingBatchKey(bID)
	exists, err := txn.Has(executingKey)
	if err != nil {
		return fmt.Errorf("check batch exists: %s", err)
	}
	if !exists {
		return fmt.Errorf("executing batch %s not found for deletion", bID)
	}

	// 1. Delete the Batch entry.
	if err := txn.Delete(executingKey); err != nil {
		return fmt.Errorf("deleting executing batch in datastore: %s", err)
	}

	// 2. Delete all related BatchableBrokerRequest entries of the batch.
	q := query.Query{
		// We query for all the batchable broker request from the batch.
		// Recall: /batch-br/<batch-id>/<batchable-broker-request-id>
		Prefix: dsPrefixBatchBrokerRequest.ChildString(string(bID)).String(),
	}
	res, err := txn.Query(q)
	if err != nil {
		return fmt.Errorf("execute bbrs for batch query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing bbrs for batch query: %s", err)
		}
	}()

	for item := range res.Next() {
		if item.Error != nil {
			return fmt.Errorf("get item result from batch query: %s", item.Error)
		}
		if err := txn.Delete(datastore.NewKey(item.Key)); err != nil {
			return fmt.Errorf("decoding gob batch batchable broker request: %s", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %s", err)
	}

	return nil
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

func save(w datastore.Write, key datastore.Key, o interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(o); err != nil {
		return fmt.Errorf("gob encoding: %s", err)
	}
	if err := w.Put(key, buf.Bytes()); err != nil {
		return fmt.Errorf("put in datastore: %s", err)
	}

	return nil
}

func getOpenBatch(w datastore.Txn, candidateSize, maxAllowedSize int64) (Batch, bool, error) {
	q := query.Query{
		Prefix: dsPrefixPendingBatch.String(),
		Orders: []dsq.Order{dsq.OrderByKey{}},
	}
	res, err := w.Query(q)
	if err != nil {
		return Batch{}, false, fmt.Errorf("creating query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing get or create open batch query: %s", err)
		}
	}()

	var b Batch
	var found bool
	for item := range res.Next() {
		if item.Error != nil {
			return Batch{}, false, fmt.Errorf("get query item result: %s", item.Error)
		}
		dec := gob.NewDecoder(bytes.NewReader(item.Value))
		if err := dec.Decode(&b); err != nil {
			return Batch{}, false, fmt.Errorf("decoding gob batch: %s", err)
		}

		if b.Size+candidateSize <= maxAllowedSize {
			found = true
			break
		}
	}
	if !found {
		return Batch{}, false, nil
	}
	log.Debugf("found open batch %s to fit data: %d %d", b.ID, b.Size, candidateSize)

	return b, true, nil
}

func makeBatchableBrokerRequestKey(bID BatchID, bbrID batchableBrokerRequestID) datastore.Key {
	return dsPrefixBatchBrokerRequest.ChildString(string(bID)).ChildString(string(bbrID))
}

func makePendingBatchKey(id BatchID) datastore.Key {
	return dsPrefixPendingBatch.ChildString(string(id))
}

func makeReadyBatchKey(id BatchID) datastore.Key {
	return dsPrefixReadyBatch.ChildString(string(id))
}

func makeExecutingBatchKey(id BatchID) datastore.Key {
	return dsPrefixExecutingBatch.ChildString(string(id))
}
