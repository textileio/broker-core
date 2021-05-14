package queue

// TODO: Restart started auctions?
// TODO: Handle retries if auction fails (got no bids / or other error).

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/gob"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	golog "github.com/ipfs/go-log/v2"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/dshelper/txndswrap"
	dsextensions "github.com/textileio/go-datastore-extensions"
)

const (
	// defaultListLimit is the default list page size.
	defaultListLimit = 10
	// maxListLimit is the max list page size.
	maxListLimit = 1000
)

var (
	log = golog.Logger("auctioneer/queue")

	// StartDelay is the time delay before the queue will process queued auctions on start.
	StartDelay = time.Second * 10

	// MaxConcurrency is the maximum number of auctions that will be handled concurrently.
	MaxConcurrency = 100

	// ErrNotFound indicates the requested auction was not found.
	ErrNotFound = errors.New("auction not found")

	// dsPrefix is the prefix for auctions.
	// Structure: /auctions/<auction_id> -> Auction.
	dsPrefix = ds.NewKey("/auctions")

	// dsQueuePrefix is the prefix for queued auctions.
	// Structure: /queue/<auction_id> -> nil.
	dsQueuePrefix = ds.NewKey("/queue")

	// dsStartedPrefix is the prefix for started auctions that are accepting bids.
	// Structure: /started/<auction_id> -> nil.
	dsStartedPrefix = ds.NewKey("/started")
)

// Handler is called when an auction moves from "queued" to "started".
// This separates the queue's job from the auction handling, making the queue logic easier to test.
type Handler func(ctx context.Context, auction *broker.Auction) error

// Queue is a persistent worker-based task queue.
type Queue struct {
	store txndswrap.TxnDatastore

	handler Handler
	jobCh   chan *broker.Auction
	doneCh  chan struct{}
	entropy *ulid.MonotonicEntropy

	ctx    context.Context
	cancel context.CancelFunc

	lk sync.Mutex
}

// NewQueue returns a new Queue using handler to process auctions.
func NewQueue(store txndswrap.TxnDatastore, handler Handler) (*Queue, error) {
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		store:   store,
		handler: handler,
		jobCh:   make(chan *broker.Auction, MaxConcurrency),
		doneCh:  make(chan struct{}, MaxConcurrency),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Create queue workers
	for i := 0; i < MaxConcurrency; i++ {
		go q.worker(i + 1)
	}

	go q.start()
	return q, nil
}

// Close the queue. This will wait for "started" auctions.
func (q *Queue) Close() error {
	q.cancel()
	return nil
}

// NewID returns new monotonically increasing auction ids.
func (q *Queue) NewID(t time.Time) (broker.AuctionID, error) {
	q.lk.Lock() // entropy is not safe for concurrent use

	if q.entropy == nil {
		q.entropy = ulid.Monotonic(rand.Reader, 0)
	}
	id, err := ulid.New(ulid.Timestamp(t.UTC()), q.entropy)
	if errors.Is(err, ulid.ErrMonotonicOverflow) {
		q.entropy = nil
		q.lk.Unlock()
		return q.NewID(t)
	} else if err != nil {
		q.lk.Unlock()
		return "", fmt.Errorf("generating id: %v", err)
	}
	q.lk.Unlock()
	return broker.AuctionID(strings.ToLower(id.String())), nil
}

// CreateAuction adds a new auction to the queue.
// The new auction will be handled immediately if workers are not busy.
func (q *Queue) CreateAuction(
	storageDealID broker.StorageDealID,
	dealSize, dealDuration uint64,
	auctionDuration time.Duration,
) (broker.AuctionID, error) {
	id, err := q.NewID(time.Now())
	if err != nil {
		return "", fmt.Errorf("creating id: %v", err)
	}
	a := &broker.Auction{
		ID:            id,
		StorageDealID: storageDealID,
		DealSize:      dealSize,
		DealDuration:  dealDuration,
		Status:        broker.AuctionStatusUnspecified,
		Duration:      auctionDuration,
	}
	if err := q.enqueue(a); err != nil {
		return "", fmt.Errorf("enqueueing: %v", err)
	}
	return id, nil
}

// GetAuction returns an auction by id.
func (q *Queue) GetAuction(id broker.AuctionID) (*broker.Auction, error) {
	a, err := getAuction(q.store, id)
	if err != nil {
		return nil, err
	}
	return a, err
}

func getAuction(reader ds.Read, id broker.AuctionID) (*broker.Auction, error) {
	val, err := reader.Get(dsPrefix.ChildString(string(id)))
	if errors.Is(err, ds.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("getting key: %v", err)
	}
	r, err := decode(val)
	if err != nil {
		return nil, fmt.Errorf("decoding value: %v", err)
	}
	return &r, nil
}

// Query is used to query for auctions.
type Query struct {
	Offset string
	Order  Order
	Limit  int
}

func (q Query) setDefaults() Query {
	if q.Limit == -1 {
		q.Limit = maxListLimit
	} else if q.Limit <= 0 {
		q.Limit = defaultListLimit
	} else if q.Limit > maxListLimit {
		q.Limit = maxListLimit
	}
	return q
}

// Order specifies the order of list results.
// Default is decending by time created.
type Order int

const (
	// OrderDescending orders results decending.
	OrderDescending Order = iota
	// OrderAscending orders results ascending.
	OrderAscending
)

// ListAuctions lists auctions by applying a Query.
func (q *Queue) ListAuctions(query Query) ([]broker.Auction, error) {
	query = query.setDefaults()

	var (
		order dsq.Order
		seek  string
		limit = query.Limit
	)

	if len(query.Offset) != 0 {
		seek = dsPrefix.ChildString(query.Offset).String()
		limit++
	}

	switch query.Order {
	case OrderDescending:
		order = dsq.OrderByKeyDescending{}
		if len(seek) == 0 {
			// Seek to largest possible key and decend from there
			seek = dsPrefix.ChildString(
				strings.ToLower(ulid.MustNew(ulid.MaxTime(), nil).String())).String()
		}
	case OrderAscending:
		order = dsq.OrderByKey{}
	}

	results, err := q.store.QueryExtended(dsextensions.QueryExt{
		Query: dsq.Query{
			Prefix: dsPrefix.String(),
			Orders: []dsq.Order{order},
			Limit:  limit,
		},
		SeekPrefix: seek,
	})
	if err != nil {
		return nil, fmt.Errorf("querying requests: %v", err)
	}
	defer func() { _ = results.Close() }()

	var list []broker.Auction
	for res := range results.Next() {
		if res.Error != nil {
			return nil, fmt.Errorf("getting next result: %v", res.Error)
		}
		a, err := decode(res.Value)
		if err != nil {
			return nil, fmt.Errorf("decoding value: %v", err)
		}
		list = append(list, a)
	}

	// Remove seek from list
	if len(query.Offset) != 0 && len(list) > 0 {
		list = list[1:]
	}

	return list, nil
}

func (q *Queue) enqueue(a *broker.Auction) error {
	// Set the auction to "started"
	if err := q.setStatus(a, broker.AuctionStatusStarted); err != nil {
		return fmt.Errorf("updating status (started): %v", err)
	}

	// Unblock the caller by letting the rest happen in the background
	go func() {
		select {
		case q.jobCh <- a:
		default:
			log.Debugf("workers are busy; queueing %s", a.ID)
			// Workers are busy, set back to "queued"
			if err := q.setStatus(a, broker.AuctionStatusQueued); err != nil {
				log.Errorf("error updating status (queued): %v", err)
			}
		}
	}()
	return nil
}

func (q *Queue) worker(num int) {
	for {
		select {
		case <-q.ctx.Done():
			return

		case a := <-q.jobCh:
			if q.ctx.Err() != nil {
				return
			}
			log.Debugf("worker %d got job %s", num, a.ID)

			// Handle the auction with the handler func
			status := broker.AuctionStatusEnded
			if err := q.handler(q.ctx, a); err != nil {
				status = broker.AuctionStatusError
				a.Error = err.Error()
				log.Errorf("error handling auction: %v", err)
			}

			// Finalize auction by setting status to "closed" or "error"
			if err := q.setStatus(a, status); err != nil {
				log.Errorf("error updating status (%s): %v", status, err)
			}

			log.Debugf("worker %d finished job %s", num, a.ID)
			go func() {
				q.doneCh <- struct{}{}
			}()
		}
	}
}

func (q *Queue) start() {
	t := time.NewTimer(StartDelay)
	for {
		select {
		case <-q.ctx.Done():
			t.Stop()
			return
		case <-t.C:
			q.getNext()
		case <-q.doneCh:
			q.getNext()
		}
	}
}

func (q *Queue) getNext() {
	a, err := q.getQueued()
	if err != nil {
		log.Errorf("getting next in queue: %v", err)
		return
	}
	if a == nil {
		return
	}
	log.Debug("enqueueing job: %s", a.ID)
	if err := q.enqueue(a); err != nil {
		log.Debugf("error enqueueing: %v", err)
	}
}

func (q *Queue) getQueued() (*broker.Auction, error) {
	txn, err := q.store.NewTransaction(true)
	if err != nil {
		return nil, fmt.Errorf("creating txn: %v", err)
	}
	defer txn.Discard()

	results, err := txn.Query(dsq.Query{
		Prefix:   dsQueuePrefix.String(),
		Orders:   []dsq.Order{dsq.OrderByKey{}},
		Limit:    1,
		KeysOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("querying queue: %v", err)
	}
	defer func() { _ = results.Close() }()

	res, ok := <-results.Next()
	if !ok {
		return nil, nil
	} else if res.Error != nil {
		return nil, fmt.Errorf("getting next result: %v", res.Error)
	}

	a, err := getAuction(txn, broker.AuctionID(path.Base(res.Key)))
	if err != nil {
		return nil, fmt.Errorf("getting auction: %v", err)
	}
	return a, nil
}

func (q *Queue) setStatus(a *broker.Auction, status broker.AuctionStatus) error {
	txn, err := q.store.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %v", err)
	}
	defer txn.Discard()

	// Handle currently "queued" and "started" status
	if a.Status == broker.AuctionStatusQueued {
		if err := txn.Delete(dsQueuePrefix.ChildString(string(a.ID))); err != nil {
			return fmt.Errorf("deleting from queue: %v", err)
		}
	} else if a.Status == broker.AuctionStatusStarted {
		if err := txn.Delete(dsStartedPrefix.ChildString(string(a.ID))); err != nil {
			return fmt.Errorf("deleting from started: %v", err)
		}
	}
	if status == broker.AuctionStatusQueued {
		a.StartedAt = time.Time{} // reset started
		if err := txn.Put(dsQueuePrefix.ChildString(string(a.ID)), nil); err != nil {
			return fmt.Errorf("putting to queue: %v", err)
		}
	} else if status == broker.AuctionStatusStarted {
		a.StartedAt = time.Now()
		if err := txn.Put(dsStartedPrefix.ChildString(string(a.ID)), nil); err != nil {
			return fmt.Errorf("putting to started: %v", err)
		}
	}

	// Update status
	a.Status = status
	val, err := encode(a)
	if err != nil {
		return fmt.Errorf("encoding value: %v", err)
	}

	// Set value
	if err := txn.Put(dsPrefix.ChildString(string(a.ID)), val); err != nil {
		return fmt.Errorf("putting value: %v", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %v", err)
	}
	return nil
}

func encode(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decode(v []byte) (a broker.Auction, err error) {
	var buf bytes.Buffer
	if _, err := buf.Write(v); err != nil {
		return a, err
	}
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&a); err != nil {
		return a, err
	}
	return a, nil
}