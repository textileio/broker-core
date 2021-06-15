package store

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

var (
	// Namespace "/unpin/pending/<cid>" contains a pending job to unpin the Cid from the ipfs layer.
	prefixPendingUnpinJob = datastore.NewKey("/unpin/pending")
	// Namespace "/unpin/executing/<cid>" contains an executing job to unpin the Cid from the ipfs layer.
	prefixExecutingUnpinJob = datastore.NewKey("/unpin/executing")
	// Namespace "/unpin/done/<cid>" contains a finalized (success|error) pin from the ipfs layer.
	prefixFinalizedUnpinJob = datastore.NewKey("/unpin/finalized")
)

type UnpinJobID string
type UnpinType int

const (
	UnpinTypeBatch UnpinType = iota
	UnpinTypeData
)

type UnpinJob struct {
	ID         UnpinJobID
	Cid        cid.Cid
	Type       UnpinType
	ReadyAt    time.Time
	ErrorCause string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// UnpinJobGetNext returns the next pending unpin job to execute.
func (s *Store) UnpinJobGetNext() (UnpinJob, bool, error) {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return UnpinJob{}, false, fmt.Errorf("creating txn: %s", err)
	}
	defer txn.Discard()

	q := query.Query{
		Prefix: prefixPendingUnpinJob.String(),
	}
	res, err := txn.Query(q)
	if err != nil {
		return UnpinJob{}, false, fmt.Errorf("executing query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing query: %s", err)
		}
	}()

	var (
		now   = time.Now()
		uj    UnpinJob
		found bool
	)
	for item := range res.Next() {
		if item.Error != nil {
			return UnpinJob{}, false, fmt.Errorf("get item result from query: %s", item.Error)
		}
		d := gob.NewDecoder(bytes.NewReader(item.Value))
		if err := d.Decode(&uj); err != nil {
			return UnpinJob{}, false, fmt.Errorf("unmarshaling gob: %s", err)
		}
		if uj.ReadyAt.Before(now) {
			found = true
			break
		}
	}
	if !found {
		return UnpinJob{}, false, nil
	}

	pendingKey := keyPendingUnpinJob(uj.ID)
	if err := txn.Delete(pendingKey); err != nil {
		return UnpinJob{}, false, fmt.Errorf("removing from pending namespace: %s", err)
	}

	uj.UpdatedAt = now
	inprogressKey := keyExecutingUnpinJob(uj.ID)
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(uj); err != nil {
		return UnpinJob{}, false, fmt.Errorf("gob encoding: %s", err)
	}
	if err := txn.Put(inprogressKey, buf.Bytes()); err != nil {
		return UnpinJob{}, false, fmt.Errorf("put in datastore: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return UnpinJob{}, false, fmt.Errorf("committing txn: %s", err)
	}

	return uj, true, nil

}

// DeleteExecuting removes an executing unpin job.
func (s *Store) DeleteExecuting(id UnpinJobID) error {
	key := keyExecutingUnpinJob(id)
	if err := s.ds.Delete(key); err != nil {
		return fmt.Errorf("deleting executing unpin job: %s", err)
	}
	return nil
}

// UnpinJobMoveToPending moves an unpin job to pending status.
func (s *Store) UnpinJobMoveToPending(id UnpinJobID, delay time.Duration) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %s", err)
	}
	defer txn.Discard()

	executingKey := keyExecutingUnpinJob(id)
	buf, err := txn.Get(executingKey)
	if err != nil {
		return fmt.Errorf("get executing unpin job: %s", err)
	}
	var uj UnpinJob
	d := gob.NewDecoder(bytes.NewReader(buf))
	if err := d.Decode(&uj); err != nil {
		return fmt.Errorf("unmarshaling gob: %s", err)
	}

	if err := txn.Delete(executingKey); err != nil {
		return fmt.Errorf("removing from executing namespace: %s", err)
	}

	uj.ReadyAt = time.Now().Add(delay)
	uj.UpdatedAt = time.Now()

	pendingKey := keyPendingUnpinJob(uj.ID)
	var buf2 bytes.Buffer
	if err := gob.NewEncoder(&buf2).Encode(uj); err != nil {
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

func keyPendingUnpinJob(id UnpinJobID) datastore.Key {
	return prefixPendingUnpinJob.ChildString(string(id))
}

func keyExecutingUnpinJob(id UnpinJobID) datastore.Key {
	return prefixExecutingUnpinJob.ChildString(string(id))
}

func keyFinalizedUnpinJob(id UnpinJobID) datastore.Key {
	return prefixFinalizedUnpinJob.ChildString(string(id))
}
