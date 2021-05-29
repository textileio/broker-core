package queue

// TODO: Restart started auctions?

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

	"github.com/ipfs/go-cid"
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

	// ErrAuctionNotFound indicates the requested auction was not found.
	ErrAuctionNotFound = errors.New("auction not found")

	// ErrBidNotFound indicates the requested bid was not found.
	ErrBidNotFound = errors.New("bid not found")

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

// Runner is called when an auction moves from "queued" to "started".
type Runner func(
	ctx context.Context,
	auction broker.Auction,
	addBid func(bid broker.Bid) (broker.BidID, error),
) (map[broker.BidID]broker.WinningBid, error)

// Finalizer is called when an auction moves from "started" to "ended" or "error".
type Finalizer func(ctx context.Context, auction broker.Auction) error

// Queue is a persistent worker-based task queue.
type Queue struct {
	store txndswrap.TxnDatastore

	runner    Runner
	finalizer Finalizer
	jobCh     chan *broker.Auction
	tickCh    chan struct{}
	entropy   *ulid.MonotonicEntropy

	runAttempts uint32

	ctx    context.Context
	cancel context.CancelFunc

	lk sync.Mutex
}

// NewQueue returns a new Queue using handler to process auctions.
func NewQueue(store txndswrap.TxnDatastore, runner Runner, finalizer Finalizer, runAttempts uint32) (*Queue, error) {
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		store:       store,
		runner:      runner,
		finalizer:   finalizer,
		jobCh:       make(chan *broker.Auction, MaxConcurrency),
		tickCh:      make(chan struct{}, MaxConcurrency),
		runAttempts: runAttempts,
		ctx:         ctx,
		cancel:      cancel,
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

// CreateAuction adds a new auction to the queue.
// The new auction will be handled immediately if workers are not busy.
func (q *Queue) CreateAuction(auction broker.Auction) (broker.AuctionID, error) {
	if err := validate(auction); err != nil {
		return "", fmt.Errorf("invalid auction data: %s", err)
	}
	id, err := q.newID(time.Now())
	if err != nil {
		return "", fmt.Errorf("creating id: %v", err)
	}
	auction.ID = id
	if err := q.enqueue(&auction); err != nil {
		return "", fmt.Errorf("enqueueing: %v", err)
	}
	return id, nil
}

func validate(a broker.Auction) error {
	if a.StorageDealID == "" {
		return errors.New("storage deal id is empty")
	}
	if a.DealSize == 0 {
		return errors.New("deal size must be greater than zero")
	}
	if a.DealDuration == 0 {
		return errors.New("deal duration must be greater than zero")
	}
	if a.DealReplication == 0 {
		return errors.New("deal replication must be greater than zero")
	}
	if a.Status != broker.AuctionStatusUnspecified {
		return errors.New("invalid initial auction status")
	}
	return nil
}

// newID returns new monotonically increasing auction ids.
func (q *Queue) newID(t time.Time) (broker.AuctionID, error) {
	q.lk.Lock() // entropy is not safe for concurrent use

	if q.entropy == nil {
		q.entropy = ulid.Monotonic(rand.Reader, 0)
	}
	id, err := ulid.New(ulid.Timestamp(t.UTC()), q.entropy)
	if errors.Is(err, ulid.ErrMonotonicOverflow) {
		q.entropy = nil
		q.lk.Unlock()
		return q.newID(t)
	} else if err != nil {
		q.lk.Unlock()
		return "", fmt.Errorf("generating id: %v", err)
	}
	q.lk.Unlock()
	return broker.AuctionID(strings.ToLower(id.String())), nil
}

// GetAuction returns an auction by id.
func (q *Queue) GetAuction(id broker.AuctionID) (*broker.Auction, error) {
	a, err := getAuction(q.store, id)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func getAuction(reader ds.Read, id broker.AuctionID) (*broker.Auction, error) {
	val, err := reader.Get(dsPrefix.ChildString(string(id)))
	if errors.Is(err, ds.ErrNotFound) {
		return nil, ErrAuctionNotFound
	} else if err != nil {
		return nil, fmt.Errorf("getting key: %v", err)
	}
	r, err := decode(val)
	if err != nil {
		return nil, fmt.Errorf("decoding value: %v", err)
	}
	return &r, nil
}

// SetWinningBidProposalCid sets the proposal cid.Cid for a broker.WinningBid, and re-queues the auction.
func (q *Queue) SetWinningBidProposalCid(id broker.AuctionID, bid broker.BidID, pcid cid.Cid) error {
	a, err := getAuction(q.store, id)
	if err != nil {
		return err
	}
	if a.Status != broker.AuctionStatusEnded {
		return errors.New("auction has not ended")
	}

	wb, ok := a.WinningBids[bid]
	if !ok {
		return ErrBidNotFound
	}
	if wb.ProposalCid.Defined() {
		return errors.New("proposal cid has already been set")
	}
	wb.ProposalCid = pcid
	a.WinningBids[bid] = wb

	allProposalsReady := true
	for _, wb := range a.WinningBids {
		if !wb.ProposalCid.Defined() {
			allProposalsReady = false
			break // wait for all proposal cids before re-queueing
		}
	}

	if allProposalsReady {
		if err := q.enqueue(a); err != nil {
			return fmt.Errorf("enqueueing: %v", err)
		}
	} else {
		// Save auction state
		if err := q.saveAndTransitionStatus(nil, a, a.Status); err != nil {
			return fmt.Errorf("saving proposal cid: %v", err)
		}
	}
	return nil
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
	if err := q.saveAndTransitionStatus(nil, a, broker.AuctionStatusStarted); err != nil {
		return fmt.Errorf("updating status (started): %v", err)
	}

	// Unblock the caller by letting the rest happen in the background
	go func() {
		select {
		case q.jobCh <- a:
		default:
			log.Debugf("workers are busy; queueing %s", a.ID)
			// Workers are busy, set back to "queued"
			if err := q.saveAndTransitionStatus(nil, a, broker.AuctionStatusQueued); err != nil {
				log.Errorf("updating status (queued): %v", err)
			}
		}
	}()
	return nil
}

func (q *Queue) worker(num int) {
	addBid := func(a *broker.Auction, bid broker.Bid) (broker.BidID, error) {
		if a.Status != broker.AuctionStatusStarted {
			return "", errors.New("auction has not started")
		}
		id, err := q.newID(bid.ReceivedAt)
		if err != nil {
			return "", fmt.Errorf("generating bid id: %v", err)
		}
		if a.Bids == nil {
			a.Bids = make(map[broker.BidID]broker.Bid)
		}
		a.Bids[broker.BidID(id)] = bid

		// Save auction data
		if err := q.saveAndTransitionStatus(nil, a, a.Status); err != nil {
			return "", fmt.Errorf("saving bid: %v", err)
		}
		return broker.BidID(id), nil
	}

	fail := func(a *broker.Auction, err error) (status broker.AuctionStatus) {
		a.Error = err.Error()
		if a.Attempts >= q.runAttempts {
			status = broker.AuctionStatusError
			log.Warnf("job %s exhausted all %d attempts with error: %v", a.ID, q.runAttempts, err)
		} else {
			status = broker.AuctionStatusQueued
			log.Debugf("retrying job %s with error: %v", a.ID, err)
		}
		return status
	}

	for {
		select {
		case <-q.ctx.Done():
			return

		case a := <-q.jobCh:
			if q.ctx.Err() != nil {
				return
			}
			a.Attempts++
			log.Debugf("worker %d got job %s (attempt=%d/%d)", num, a.ID, a.Attempts, q.runAttempts)

			// Handle the auction with the runner func
			var status broker.AuctionStatus
			wbs, err := q.runner(q.ctx, *a, func(bid broker.Bid) (broker.BidID, error) {
				return addBid(a, bid)
			})
			// Update winning bid state; some bids may have been processed even if there was an error
			if len(wbs) > 0 {
				if a.WinningBids == nil {
					a.WinningBids = make(map[broker.BidID]broker.WinningBid)
				}
				for id, wb := range wbs {
					a.WinningBids[id] = wb
				}
			}
			if err != nil {
				status = fail(a, err)
			} else {
				status = broker.AuctionStatusEnded
				// Reset error
				a.Error = ""
			}

			// Save and update status to "ended" or "error"
			if err := q.saveAndTransitionStatus(nil, a, status); err != nil {
				log.Errorf("updating runner status (%s): %v", status, err)
			} else if status != broker.AuctionStatusQueued {
				// Finish auction with the finalizer func
				if err := q.finalizer(q.ctx, *a); err != nil {
					status = fail(a, err)

					// Save and update status to "queued" or "error"
					if err := q.saveAndTransitionStatus(nil, a, status); err != nil {
						log.Errorf("updating finalizer status (%s): %v", status, err)
					}
				}
			}

			log.Debugf("worker %d finished job %s", num, a.ID)
			go func() {
				q.tickCh <- struct{}{}
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
		case <-q.tickCh:
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
	log.Debugf("enqueueing job: %s", a.ID)
	if err := q.enqueue(a); err != nil {
		log.Debugf("error enqueueing: %v", err)
	}
}

func (q *Queue) getQueued() (*broker.Auction, error) {
	txn, err := q.store.NewTransaction(false)
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
	if err := q.saveAndTransitionStatus(txn, a, broker.AuctionStatusStarted); err != nil {
		return nil, fmt.Errorf("getting auction: %v", err)
	}
	return a, nil
}

// saveAndTransitionStatus sets a new status, updating the started time if needed.
// Do not directly edit the auction status because it is needed to determine the correct status transition.
// Pass the desired new status with newStatus.
func (q *Queue) saveAndTransitionStatus(txn ds.Txn, a *broker.Auction, newStatus broker.AuctionStatus) error {
	if txn == nil {
		var err error
		txn, err = q.store.NewTransaction(false)
		if err != nil {
			return fmt.Errorf("creating txn: %v", err)
		}
		defer txn.Discard()
	}

	if a.Status != newStatus {
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
		if newStatus == broker.AuctionStatusQueued {
			a.StartedAt = time.Time{} // reset started
			if err := txn.Put(dsQueuePrefix.ChildString(string(a.ID)), nil); err != nil {
				return fmt.Errorf("putting to queue: %v", err)
			}
		} else if newStatus == broker.AuctionStatusStarted {
			a.StartedAt = time.Now()
			if err := txn.Put(dsStartedPrefix.ChildString(string(a.ID)), nil); err != nil {
				return fmt.Errorf("putting to started: %v", err)
			}
		}
		// Update status
		a.Status = newStatus
	}

	val, err := encode(a)
	if err != nil {
		return fmt.Errorf("encoding value: %v", err)
	}
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
