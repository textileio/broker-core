package queue

// @todo: Re-handle "open" auctions

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
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/oklog/ulid/v2"
	kt "github.com/textileio/broker-core/keytransform"
	dsextensions "github.com/textileio/go-datastore-extensions"
)

const (
	LogName = "auctioneer/queue"

	// defaultListLimit is the default list page size.
	defaultListLimit = 10
	// maxListLimit is the max list page size.
	maxListLimit = 1000
)

var (
	log = golog.Logger(LogName)

	// StartDelay is the time delay before the queue will process queued auctions on start.
	StartDelay = time.Second * 10

	// MaxConcurrency is the maximum number of auctions that will be handled concurrently.
	MaxConcurrency = 100

	// ErrNotFound indicates the requested auction was not found.
	ErrNotFound = errors.New("auction not found")

	// ErrInProgress indicates the auction is in progress and cannot be altered.
	ErrInProgress = errors.New("auction in progress")

	// dsPrefix is the prefix for auctions.
	// Structure: /auctions/<auction_id> -> Auction
	dsPrefix = ds.NewKey("/auctions")

	// dsQueuePrefix is the prefix for queued auctions.
	// Structure: /queue/<auction_id> -> nil
	dsQueuePrefix = ds.NewKey("/queue")

	// dsOpenPrefix is the prefix for open auctions that are accepting bids.
	// Structure: /open/<auction_id> -> nil
	dsOpenPrefix = ds.NewKey("/open")

	// dsBidPrefix is the prefix for bids grouped by auction.
	// Structure: /auctions/<auction_id>/bids/<bid_id> -> Bid
	dsBidPrefix = ds.NewKey("/bids")
)

type Status string

const (
	StatusNew    Status = "new"
	StatusQueued Status = "queued"
	StatusOpen   Status = "open"
	StatusClosed Status = "closed"
	StatusError  Status = "error"
)

// Auction is the persisted auction model.
type Auction struct {
	ID        string
	Status    Status
	Winner    peer.ID
	StartedAt time.Time
	EndedAt   time.Time
	Error     string
}

// Handler is called when an auction moves from "queued" to "open".
// This separates the queue's job from the auction handling, making the queue logic easier to test.
type Handler func(ctx context.Context, auction Auction) error

// Queue is a persistent worker-based task queue.
type Queue struct {
	store kt.TxnDatastoreExtended

	handler Handler
	jobCh   chan Auction
	doneCh  chan struct{}
	entropy *ulid.MonotonicEntropy

	ctx    context.Context
	cancel context.CancelFunc

	lk sync.Mutex
}

// NewQueue returns a new Queue using handler to process auctions.
func NewQueue(store kt.TxnDatastoreExtended, handler Handler) (*Queue, error) {
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		store:   store,
		handler: handler,
		jobCh:   make(chan Auction, MaxConcurrency),
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

// Close the queue and cancel "open" auctions.
func (q *Queue) Close() error {
	q.cancel()
	return nil
}

// NewID returns new monotonically increasing auction ids.
func (q *Queue) NewID(t time.Time) (string, error) {
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
	return strings.ToLower(id.String()), nil
}

// CreateAuction adds a new auction to the queue.
// The new auction will be handled immediately if workers are not busy.
func (q *Queue) CreateAuction() (*Auction, error) {
	id, err := q.NewID(time.Now())
	if err != nil {
		return nil, fmt.Errorf("creating id: %v", err)
	}
	a := Auction{
		ID:     id,
		Status: StatusNew,
	}
	if err := q.enqueue(a); err != nil {
		return nil, fmt.Errorf("enqueueing: %v", err)
	}
	return &a, nil
}

// GetAuction returns an auction by id.
func (q *Queue) GetAuction(id string) (*Auction, error) {
	a, err := q.getAuction(q.store, id)
	if err != nil {
		return nil, err
	}
	return a, err
}

func (q *Queue) getAuction(reader ds.Read, id string) (*Auction, error) {
	val, err := reader.Get(dsPrefix.ChildString(id))
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
	OrderDescending Order = iota
	OrderAscending
)

// ListAuctions lists auctions by applying a Query.
func (q *Queue) ListAuctions(query Query) ([]Auction, error) {
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
	defer results.Close()

	var list []Auction
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

func (q *Queue) enqueue(a Auction) error {
	// Set the auction to "open"
	if err := q.setStatus(a, StatusOpen); err != nil {
		return fmt.Errorf("updating status (open): %v", err)
	}

	// Unblock the caller by letting the rest happen in the background
	go func() {
		select {
		case q.jobCh <- a:
		default:
			log.Debugf("workers are busy; queueing %s", a.ID)
			// Workers are busy, set back to "queued"
			if err := q.setStatus(a, StatusQueued); err != nil {
				log.Debugf("error updating status (queued): %v", err)
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
			a.StartedAt = time.Now()
			status := StatusClosed
			if err := q.handler(q.ctx, a); err != nil {
				status = StatusError
				a.Error = err.Error()
				log.Debugf("error handling auction: %v", err)
			}
			a.EndedAt = time.Now()

			// Finalize auction by setting status to "closed" or "error"
			if err := q.setStatus(a, status); err != nil {
				log.Debugf("error updating status (%s): %v", status, err)
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
	if err := q.enqueue(*a); err != nil {
		log.Debugf("error enqueueing: %v", err)
	}
}

func (q *Queue) getQueued() (*Auction, error) {
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
	defer results.Close()

	var key string
	for res := range results.Next() {
		if res.Error != nil {
			return nil, fmt.Errorf("getting next result: %v", res.Error)
		}
		key = res.Key
		break
	}
	if len(key) == 0 {
		return nil, nil
	}

	a, err := q.getAuction(txn, path.Base(key))
	if err != nil {
		return nil, fmt.Errorf("getting auction: %v", err)
	}
	return a, nil
}

func (q *Queue) setStatus(a Auction, status Status) error {
	txn, err := q.store.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %v", err)
	}
	defer txn.Discard()

	// Handle currently "queued" and "open" status
	if a.Status == StatusQueued {
		if err := txn.Delete(dsQueuePrefix.ChildString(a.ID)); err != nil {
			return fmt.Errorf("deleting from queue: %v", err)
		}
	} else if a.Status == StatusOpen {
		if err := txn.Delete(dsOpenPrefix.ChildString(a.ID)); err != nil {
			return fmt.Errorf("deleting from open: %v", err)
		}
	}
	if status == StatusQueued {
		if err := txn.Put(dsQueuePrefix.ChildString(a.ID), nil); err != nil {
			return fmt.Errorf("putting to queue: %v", err)
		}
	} else if status == StatusOpen {
		if err := txn.Put(dsOpenPrefix.ChildString(a.ID), nil); err != nil {
			return fmt.Errorf("putting to open: %v", err)
		}
	}

	// Update status
	a.Status = status
	val, err := encode(a)
	if err != nil {
		return fmt.Errorf("encoding value: %v", err)
	}

	// Set value
	if err := txn.Put(dsPrefix.ChildString(a.ID), val); err != nil {
		return fmt.Errorf("putting value: %v", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %v", err)
	}
	return nil
}

func encode(a Auction) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(a); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decode(v []byte) (a Auction, err error) {
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
