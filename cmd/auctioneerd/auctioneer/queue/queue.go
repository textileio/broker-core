package queue

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
	"github.com/oklog/ulid/v2"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/dshelper/txndswrap"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	dsextensions "github.com/textileio/go-datastore-extensions"
	golog "github.com/textileio/go-log/v2"
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
	MaxConcurrency = 1

	// ErrAuctionNotFound indicates the requested auction was not found.
	ErrAuctionNotFound = errors.New("auction not found")

	// ErrBidNotFound indicates the requested bid was not found.
	ErrBidNotFound = errors.New("bid not found")

	// dsPrefix is the prefix for auctions.
	// Structure: /auctions/<auction_id> -> Auction.
	dsPrefix = ds.NewKey("/auctions")

	// dsQueuedPrefix is the prefix for queued auctions.
	// Structure: /queued/<auction_id> -> nil.
	dsQueuedPrefix = ds.NewKey("/queued")

	// dsStartedPrefix is the prefix for started auctions that are accepting bids.
	// Structure: /started/<auction_id> -> nil.
	dsStartedPrefix = ds.NewKey("/started")
)

// Handler is called when an auction moves from "queued" to "started".
type Handler func(
	ctx context.Context,
	auction auctioneer.Auction,
	addBid func(bid auctioneer.Bid) (auction.BidID, error),
) (map[auction.BidID]auctioneer.WinningBid, error)

// Finalizer is called when an auction moves from "started" to "finalized".
type Finalizer func(ctx context.Context, auction *auctioneer.Auction) error

// Queue is a persistent worker-based task queue.
type Queue struct {
	store txndswrap.TxnDatastore

	handler   Handler
	finalizer Finalizer
	jobCh     chan *auctioneer.Auction
	tickCh    chan struct{}
	entropy   *ulid.MonotonicEntropy

	ctx    context.Context
	cancel context.CancelFunc

	wg sync.WaitGroup
	lk sync.Mutex
}

// NewQueue returns a new Queue using handler to process auctions.
func NewQueue(store txndswrap.TxnDatastore, handler Handler, finalizer Finalizer) *Queue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		store:     store,
		handler:   handler,
		finalizer: finalizer,
		jobCh:     make(chan *auctioneer.Auction, MaxConcurrency),
		tickCh:    make(chan struct{}, MaxConcurrency),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Create queue workers
	q.wg.Add(MaxConcurrency)
	for i := 0; i < MaxConcurrency; i++ {
		go q.worker(i + 1)
	}

	go q.start()
	return q
}

// Close the queue. This will wait for "started" auctions.
func (q *Queue) Close() error {
	q.cancel()
	q.wg.Wait()
	return nil
}

// CreateAuction adds a new auction to the queue.
// The new auction will be handled immediately if workers are not busy.
func (q *Queue) CreateAuction(auction auctioneer.Auction) error {
	if err := validate(auction); err != nil {
		return fmt.Errorf("invalid auction data: %s", err)
	}
	if err := q.enqueue(nil, &auction); err != nil {
		return fmt.Errorf("enqueueing: %v", err)
	}

	log.Debugf("created auction %s", auction.ID)
	return nil
}

func validate(a auctioneer.Auction) error {
	if a.ID == "" {
		return errors.New("auction id is empty")
	}
	if a.BatchID == "" {
		return errors.New("batch id is empty")
	}
	if !a.PayloadCid.Defined() {
		return errors.New("payload cid is empty")
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
	if err := a.Sources.Validate(); err != nil {
		return err
	}
	if a.Status != broker.AuctionStatusUnspecified {
		return errors.New("invalid initial auction status")
	}
	if len(a.Bids) != 0 {
		return errors.New("initial bids must be empty")
	}
	if len(a.WinningBids) != 0 {
		return errors.New("initial winning bids must be empty")
	}
	if !a.StartedAt.IsZero() {
		return errors.New("initial started at must be zero")
	}
	if !a.UpdatedAt.IsZero() {
		return errors.New("initial updated at must be zero")
	}
	if a.Duration == 0 {
		return errors.New("duration must be greater than zero")
	}
	if a.ErrorCause != "" {
		return errors.New("initial error cause must be empty")
	}
	return nil
}

// newID returns new monotonically increasing auction ids.
func (q *Queue) newID(t time.Time) (auction.ID, error) {
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
	return auction.ID(strings.ToLower(id.String())), nil
}

// GetAuction returns an auction by id.
// If an auction is not found for id, ErrAuctionNotFound is returned.
func (q *Queue) GetAuction(id auction.ID) (*auctioneer.Auction, error) {
	a, err := getAuction(q.store, id)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func getAuction(reader ds.Read, id auction.ID) (*auctioneer.Auction, error) {
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

// SetWinningBidProposalCid sets the proposal cid.Cid for an auctioneer.WinningBid.
// If an auction is not found for id, ErrAuctionNotFound is returned.
// If a bid is not found for id, ErrBidNotFound is returned.
func (q *Queue) SetWinningBidProposalCid(
	id auction.ID,
	bid auction.BidID,
	pcid cid.Cid,
	handler func(wb auctioneer.WinningBid) error,
) error {
	if !pcid.Defined() {
		return errors.New("proposal cid is not defined")
	}

	txn, err := q.store.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %v", err)
	}
	defer txn.Discard()

	a, err := getAuction(txn, id)
	if err != nil {
		return err
	}
	wb, ok := a.WinningBids[bid]
	if !ok {
		return ErrBidNotFound
	}

	// Check if auction is in good standing
	if a.Status != broker.AuctionStatusFinalized {
		return errors.New("auction is not finalized")
	}
	if a.ErrorCause != "" {
		return errors.New("auction finalized with error; can't set proposal cid")
	}

	if wb.ProposalCid == pcid {
		log.Warnf("proposal cid %s is already published, duplicated message?", pcid)
		return nil
	}
	wb.ProposalCid = pcid
	handleErr := handler(wb)
	if handleErr != nil {
		wb.ProposalCid = cid.Undef
		wb.ErrorCause = handleErr.Error()
	} else {
		wb.ErrorCause = ""
	}
	a.WinningBids[bid] = wb

	if err := q.saveAndTransitionStatus(txn, a, a.Status); err != nil {
		return fmt.Errorf("saving proposal cid: %v", err)
	}
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %v", err)
	}
	return handleErr
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
func (q *Queue) ListAuctions(query Query) ([]auctioneer.Auction, error) {
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
	defer func() {
		if err := results.Close(); err != nil {
			log.Errorf("closing results: %v", err)
		}
	}()

	var list []auctioneer.Auction
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

// enqueue an auction.
// commitTxn will be committed internally!
func (q *Queue) enqueue(commitTxn ds.Txn, a *auctioneer.Auction) error {
	// Set the auction to "started"
	if err := q.saveAndTransitionStatus(commitTxn, a, broker.AuctionStatusStarted); err != nil {
		return fmt.Errorf("updating status (started): %v", err)
	}
	if commitTxn != nil {
		if err := commitTxn.Commit(); err != nil {
			return fmt.Errorf("committing txn: %v", err)
		}
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
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return

		case a := <-q.jobCh:
			if q.ctx.Err() != nil {
				return
			}
			log.Debugf("worker %d started auction %s", num, a.ID)

			// Handle the auction with the handler func
			wbs, err := q.handler(q.ctx, *a, func(bid auctioneer.Bid) (auction.BidID, error) {
				return q.addBid(a, bid)
			})
			// Update winning bid state; some bids may have been processed even if there was an error
			if len(wbs) > 0 {
				if a.WinningBids == nil {
					a.WinningBids = make(map[auction.BidID]auctioneer.WinningBid)
				}
				for id, wb := range wbs {
					a.WinningBids[id] = wb
				}
			}
			if err != nil {
				a.ErrorCause = err.Error()
				log.Debugf("auction %s failed: %s", a.ID, a.ErrorCause)
			} else {
				log.Debugf("auction %s succeeded", a.ID)
			}

			q.saveAndFinalizeAuction(a)
			select {
			case q.tickCh <- struct{}{}:
			default:
			}
		}
	}
}

func (q *Queue) saveAndFinalizeAuction(a *auctioneer.Auction) {
	// Save and update status to "finalized"
	if err := q.saveAndTransitionStatus(nil, a, broker.AuctionStatusFinalized); err != nil {
		log.Errorf("updating status (%s): %v", broker.AuctionStatusFinalized, err)
		return
	}

	// Finish auction with the finalizer func
	if err := q.finalizer(q.ctx, a); err != nil {
		a.ErrorCause = err.Error()

		// Save error
		if err := q.saveAndTransitionStatus(nil, a, a.Status); err != nil {
			log.Errorf("saving finalizer error: %v", err)
		}
	}
}

func (q *Queue) addBid(a *auctioneer.Auction, bid auctioneer.Bid) (auction.BidID, error) {
	if a.Status != broker.AuctionStatusStarted {
		return "", errors.New("auction has not started")
	}
	id, err := q.newID(bid.ReceivedAt)
	if err != nil {
		return "", fmt.Errorf("generating bid id: %v", err)
	}
	if a.Bids == nil {
		a.Bids = make(map[auction.BidID]auctioneer.Bid)
	}
	a.Bids[auction.BidID(id)] = bid

	// Save auction data
	if err := q.saveAndTransitionStatus(nil, a, a.Status); err != nil {
		return "", fmt.Errorf("saving bid: %v", err)
	}
	return auction.BidID(id), nil
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
	txn, err := q.store.NewTransaction(false)
	if err != nil {
		log.Errorf("creating txn: %v", err)
		return
	}
	defer txn.Discard()

	a, err := q.getQueued(txn)
	if err != nil {
		log.Errorf("getting next in queue: %v", err)
		return
	}
	if a == nil {
		return
	}
	log.Debugf("enqueueing job: %s", a.ID)
	if err := q.enqueue(txn, a); err != nil {
		log.Errorf("enqueueing: %v", err)
	}
}

func (q *Queue) getQueued(txn ds.Txn) (*auctioneer.Auction, error) {
	results, err := txn.Query(dsq.Query{
		Prefix:   dsQueuedPrefix.String(),
		Orders:   []dsq.Order{dsq.OrderByKey{}},
		Limit:    1,
		KeysOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("querying queue: %v", err)
	}
	defer func() {
		if err := results.Close(); err != nil {
			log.Errorf("closing results: %v", err)
		}
	}()

	res, ok := <-results.Next()
	if !ok {
		return nil, nil
	} else if res.Error != nil {
		return nil, fmt.Errorf("getting next result: %v", res.Error)
	}

	a, err := getAuction(txn, auction.ID(path.Base(res.Key)))
	if err != nil {
		return nil, fmt.Errorf("getting auction: %v", err)
	}
	return a, nil
}

// saveAndTransitionStatus sets a new status, updating the started time if needed.
// Do not directly edit the auction status because it is needed to determine the correct status transition.
// Pass the desired new status with newStatus.
func (q *Queue) saveAndTransitionStatus(txn ds.Txn, a *auctioneer.Auction,
	newStatus broker.AuctionStatus) error {
	commitTxn := txn == nil
	if commitTxn {
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
			if err := txn.Delete(dsQueuedPrefix.ChildString(string(a.ID))); err != nil {
				return fmt.Errorf("deleting from queued: %v", err)
			}
		} else if a.Status == broker.AuctionStatusStarted {
			if err := txn.Delete(dsStartedPrefix.ChildString(string(a.ID))); err != nil {
				return fmt.Errorf("deleting from started: %v", err)
			}
		}
		if newStatus == broker.AuctionStatusQueued {
			a.StartedAt = time.Time{} // reset started
			if err := txn.Put(dsQueuedPrefix.ChildString(string(a.ID)), nil); err != nil {
				return fmt.Errorf("putting to queued: %v", err)
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

	a.UpdatedAt = time.Now()
	val, err := encode(a)
	if err != nil {
		return fmt.Errorf("encoding value: %v", err)
	}
	if err := txn.Put(dsPrefix.ChildString(string(a.ID)), val); err != nil {
		return fmt.Errorf("putting value: %v", err)
	}
	if commitTxn {
		if err := txn.Commit(); err != nil {
			return fmt.Errorf("committing txn: %v", err)
		}
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

func decode(v []byte) (a auctioneer.Auction, err error) {
	dec := gob.NewDecoder(bytes.NewReader(v))
	if err := dec.Decode(&a); err != nil {
		return a, err
	}
	return a, nil
}
