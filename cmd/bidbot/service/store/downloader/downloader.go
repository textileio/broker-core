package downloader

//
// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"path"
// 	"sync"
// 	"time"
//
// 	"github.com/ipfs/go-cid"
// 	ds "github.com/ipfs/go-datastore"
// 	dsq "github.com/ipfs/go-datastore/query"
// 	golog "github.com/ipfs/go-log/v2"
// 	"github.com/textileio/broker-core/broker"
// 	"github.com/textileio/broker-core/cmd/bidbot/service/store"
// 	"github.com/textileio/broker-core/dshelper/txndswrap"
// )
//
// var (
// 	log = golog.Logger("bidbot/downloader")
//
// 	// StartDelay is the time delay before the queue will process queued downloads on start.
// 	StartDelay = time.Second * 10
//
// 	// MaxConcurrency is the maximum number of downloads that will be handled concurrently.
// 	MaxConcurrency = 10
//
// 	// ErrBidNotFound indicates the requested bid was not found.
// 	ErrBidNotFound = errors.New("bid not found")
//
// 	// dsQueuePrefix is the prefix for queued downloads.
// 	// Structure: /queue/<bid_id> -> cid.Cid.
// 	dsQueuePrefix = ds.NewKey("/queue")
//
// 	// dsStartedPrefix is the prefix for started downloads.
// 	// Structure: /started/<bid_id> -> cid.Cid.
// 	dsStartedPrefix = ds.NewKey("/started")
// )
//
// // Queue handles downloading bid proposal cids.
// type Queue struct {
// 	store txndswrap.TxnDatastore
//
// 	jobCh  chan store.Bid
// 	tickCh chan struct{}
//
// 	downloadAttempts uint32
//
// 	ctx    context.Context
// 	cancel context.CancelFunc
//
// 	lk sync.Mutex
// }
//
// // NewQueue returns a new Queue.
// func NewQueue(s txndswrap.TxnDatastore, downloadAttempts uint32) (*Queue, error) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	q := &Queue{
// 		store:            s,
// 		jobCh:            make(chan store.Bid, MaxConcurrency),
// 		tickCh:           make(chan struct{}, MaxConcurrency),
// 		downloadAttempts: downloadAttempts,
// 		ctx:              ctx,
// 		cancel:           cancel,
// 	}
//
// 	// Create queue workers
// 	for i := 0; i < MaxConcurrency; i++ {
// 		go q.worker(i + 1)
// 	}
//
// 	go q.start()
// 	return q, nil
// }
//
// // Close the store. This will wait for "started" downloads.
// func (q *Queue) Close() error {
// 	s.cancel()
// 	return nil
// }
//
// func (q *Queue) enqueue(id broker.BidID) error {
// 	// Set the bid to "started"
// 	if err := s.saveAndTransitionStatus(nil, b, BidStatusDownloadingProposalCid); err != nil {
// 		return fmt.Errorf("updating status (started): %v", err)
// 	}
//
// 	// Unblock the caller by letting the rest happen in the background
// 	go func() {
// 		select {
// 		case q.jobCh <- id:
// 		default:
// 			log.Debugf("workers are busy; queueing %s", id)
// 			// Workers are busy, set back to "queued"
// 			if err := s.saveAndTransitionStatus(nil, b, BidStatusQueuedProposalCid); err != nil {
// 				log.Errorf("updating status (queued): %v", err)
// 			}
// 		}
// 	}()
// 	return nil
// }
//
// func (q *Queue) worker(num int) {
// 	fail := func(b *Bid, err error) (status BidStatus) {
// 		b.ErrorCause = err.Error()
// 		if b.Attempts >= s.downloadAttempts {
// 			status = BidStatusError
// 			log.Warnf("job %s exhausted all %d attempts with error: %v", b.ID, s.downloadAttempts, err)
// 		} else {
// 			status = BidStatusQueuedProposalCid
// 			log.Debugf("retrying job %s with error: %v", b.ID, err)
// 		}
// 		return status
// 	}
//
// 	for {
// 		select {
// 		case <-s.ctx.Done():
// 			return
//
// 		case b := <-s.jobCh:
// 			if s.ctx.Err() != nil {
// 				return
// 			}
// 			b.Attempts++
// 			log.Debugf("worker %d got job %s (attempt=%d/%d)", num, b.ID, b.Attempts, s.downloadAttempts)
//
// 			// Handle the bid with the runner func
// 			var status BidStatus
// 			if err := s.runner(s.ctx, *b); err != nil {
// 				status = fail(b, err)
// 			} else {
// 				status = BidStatusComplete
// 				// Reset error
// 				b.ErrorCause = ""
// 			}
//
// 			// Save and update status to "ended" or "error"
// 			if err := s.saveAndTransitionStatus(nil, b, status); err != nil {
// 				log.Errorf("updating runner status (%s): %v", status, err)
// 			}
//
// 			log.Debugf("worker %d finished job %s", num, b.ID)
// 			go func() {
// 				s.tickCh <- struct{}{}
// 			}()
// 		}
// 	}
// }
//
// func (q *Queue) start() {
// 	t := time.NewTimer(StartDelay)
// 	for {
// 		select {
// 		case <-s.ctx.Done():
// 			t.Stop()
// 			return
// 		case <-t.C:
// 			s.getNext()
// 		case <-s.tickCh:
// 			s.getNext()
// 		}
// 	}
// }
//
// func (q *Queue) getNext() {
// 	id, err := q.getQueued()
// 	if err != nil {
// 		log.Errorf("getting next in queue: %v", err)
// 		return
// 	}
// 	if id == "" {
// 		return
// 	}
// 	log.Debugf("enqueueing job: %s", id)
// 	if err := q.enqueue(id); err != nil {
// 		log.Debugf("error enqueueing: %v", err)
// 	}
// }
//
// func (q *Queue) getQueued() (broker.BidID, error) {
// 	txn, err := q.store.NewTransaction(false)
// 	if err != nil {
// 		return "", fmt.Errorf("creating txn: %v", err)
// 	}
// 	defer txn.Discard()
//
// 	results, err := txn.Query(dsq.Query{
// 		Prefix: dsQueuePrefix.String(),
// 		Orders: []dsq.Order{dsq.OrderByKey{}},
// 		Limit:  1,
// 	})
// 	if err != nil {
// 		return "", fmt.Errorf("querying queue: %v", err)
// 	}
// 	defer func() { _ = results.Close() }()
//
// 	res, ok := <-results.Next()
// 	if !ok {
// 		return "", nil
// 	} else if res.Error != nil {
// 		return "", fmt.Errorf("getting next result: %v", res.Error)
// 	}
//
// 	id := broker.BidID(path.Base(res.Key))
// 	pcid, err := cid.Cast(res.Value)
// 	if err != nil {
// 		return "", fmt.Errorf("decoding cid: %v", err)
// 	}
//
// 	if err := txn.Delete(dsQueuePrefix.ChildString(string(id))); err != nil {
// 		return "", fmt.Errorf("deleting from queue: %v", err)
// 	}
// 	if err := txn.Put(dsStartedPrefix.ChildString(string(id)), pcid.Bytes()); err != nil {
// 		return "", fmt.Errorf("putting to started: %v", err)
// 	}
//
// 	if err := txn.Commit(); err != nil {
// 		return "", fmt.Errorf("committing txn: %v", err)
// 	}
// 	return id, nil
// }
//
// func (q *Queue) dd(txn ds.Txn, id broker.BidID, newStatus store.BidStatus) error {
//
// 	val, err := encode(b)
// 	if err != nil {
// 		return fmt.Errorf("encoding bid: %v", err)
// 	}
// 	if err := txn.Put(dsPrefix.ChildString(string(b.ID)), val); err != nil {
// 		return fmt.Errorf("putting bid: %v", err)
// 	}
// 	if err := txn.Commit(); err != nil {
// 		return fmt.Errorf("committing txn: %v", err)
// 	}
// 	return nil
// }
