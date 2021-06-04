package store

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	format "github.com/ipfs/go-ipld-format"
	golog "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-car"
	"github.com/libp2p/go-libp2p-core/peer"
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
	log = golog.Logger("bidbot/store")

	// ProposalFetchStartDelay is the time delay before the store will process queued proposal cids on start.
	ProposalFetchStartDelay = time.Second * 10

	// ProposalFetchTimeout is the timeout used when fetching proposal cids.
	ProposalFetchTimeout = time.Hour

	// MaxProposalFetchConcurrency is the maximum number of auctions that will be handled concurrently.
	MaxProposalFetchConcurrency = 100

	// ErrBidNotFound indicates the requested bid was not found.
	ErrBidNotFound = errors.New("bid not found")

	// dsPrefix is the prefix for auctions.
	// Structure: /auctions/<auction_id> -> Auction.
	dsPrefix = ds.NewKey("/bids")

	// dsQueuePrefix is the prefix for queued auctions.
	// Structure: /queue/<auction_id> -> nil.
	dsQueuePrefix = ds.NewKey("/queue")

	// dsStartedPrefix is the prefix for started auctions that are accepting bids.
	// Structure: /started/<auction_id> -> nil.
	dsStartedPrefix = ds.NewKey("/started")
)

// Bid defines the core bid model from a miner's perspective.
type Bid struct {
	ID                       broker.BidID
	AuctionID                broker.AuctionID
	AuctioneerID             peer.ID
	Status                   BidStatus
	AskPrice                 int64 // attoFIL per GiB per epoch
	VerifiedAskPrice         int64 // attoFIL per GiB per epoch
	StartEpoch               uint64
	FastRetrieval            bool
	ProposalCid              cid.Cid
	ProposalCidFetchAttempts uint32
	CreatedAt                time.Time
	UpdatedAt                time.Time
	ErrorCause               string
}

// BidStatus is the status of a Bid.
type BidStatus int

const (
	// BidStatusUnspecified indicates the initial or invalid status of a bid.
	BidStatusUnspecified BidStatus = iota
	// BidStatusSubmitted indicates the bid was successfully submitted to the auctioneer.
	BidStatusSubmitted
	// BidStatusAwaitingProposal indicates the bid was accepted and is awaiting proposal cid from auctioneer.
	BidStatusAwaitingProposal
	// BidStatusQueuedProposal indicates the bid proposal cid was received but downloading is queued.
	BidStatusQueuedProposal
	// BidStatusFetchingProposal indicates the bid proposal cid is being fetched.
	BidStatusFetchingProposal
	// BidStatusFinalized indicates the bid has reached a final state.
	// If ErrorCause is empty, the bid has been accepted and the proposal cid downloaded.
	// If ErrorCause is not empty, a fatal error has occurred and the bid should be considered abandoned.
	BidStatusFinalized
)

// String returns a string-encoded status.
func (as BidStatus) String() string {
	switch as {
	case BidStatusUnspecified:
		return "unspecified"
	case BidStatusSubmitted:
		return "submitted"
	case BidStatusAwaitingProposal:
		return "awaiting_proposal"
	case BidStatusQueuedProposal:
		return "queued_proposal"
	case BidStatusFetchingProposal:
		return "fetching_proposal"
	case BidStatusFinalized:
		return "finalized"
	default:
		return "invalid"
	}
}

// Store stores miner auction deal bids.
type Store struct {
	store      txndswrap.TxnDatastore
	nodeGetter format.NodeGetter

	jobCh  chan *Bid
	tickCh chan struct{}

	proposalDataDirectory string
	proposalFetchAttempts uint32

	ctx    context.Context
	cancel context.CancelFunc
}

// NewStore returns a new Store using handler to process auctions.
func NewStore(
	store txndswrap.TxnDatastore,
	nodeGetter format.NodeGetter,
	proposalDataDirectory string,
	proposalFetchAttempts uint32,
) (*Store, error) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Store{
		store:                 store,
		nodeGetter:            nodeGetter,
		jobCh:                 make(chan *Bid, MaxProposalFetchConcurrency),
		tickCh:                make(chan struct{}, MaxProposalFetchConcurrency),
		proposalDataDirectory: proposalDataDirectory,
		proposalFetchAttempts: proposalFetchAttempts,
		ctx:                   ctx,
		cancel:                cancel,
	}

	// Create proposal fetch workers
	for i := 0; i < MaxProposalFetchConcurrency; i++ {
		go s.fetchWorker(i + 1)
	}

	// Re-enqueue jobs that may have been orphaned during an forced shutdown
	if err := s.getOrphaned(); err != nil {
		return nil, fmt.Errorf("getting orphaned jobs: %v", err)
	}

	go s.startFetching()
	return s, nil
}

// Close the store. This will wait for "started" auctions.
func (s *Store) Close() error {
	s.cancel()
	return nil
}

// SaveBid saves a bid that has been submitted to an auctioneer.
func (s *Store) SaveBid(bid Bid) error {
	if err := validate(bid); err != nil {
		return fmt.Errorf("invalid bid data: %s", err)
	}

	bid.CreatedAt = time.Now()
	if err := s.saveAndTransitionStatus(nil, &bid, BidStatusSubmitted); err != nil {
		return fmt.Errorf("saving bid: %v", err)
	}
	return nil
}

func validate(b Bid) error {
	if b.ID == "" {
		return errors.New("id is empty")
	}
	if b.AuctionID == "" {
		return errors.New("auction id is empty")
	}
	if b.AuctioneerID.Validate() != nil {
		return errors.New("auctioneer id is not a valid peer id")
	}
	if b.Status != BidStatusUnspecified {
		return errors.New("invalid initial bid status")
	}
	if b.AskPrice < 0 {
		return errors.New("ask price must be greater than or equal to zero")
	}
	if b.VerifiedAskPrice < 0 {
		return errors.New("verified ask price must be greater than or equal to zero")
	}
	if b.StartEpoch <= 0 {
		return errors.New("start epoch must be greater than zero")
	}
	if b.ProposalCid.Defined() {
		return errors.New("initial proposal cid cannot be defined")
	}
	if b.ProposalCidFetchAttempts != 0 {
		return errors.New("initial proposal cid download attempts must be zero")
	}
	if !b.CreatedAt.IsZero() {
		return errors.New("initial created at must be zero")
	}
	if !b.UpdatedAt.IsZero() {
		return errors.New("initial updated at must be zero")
	}
	if b.ErrorCause != "" {
		return errors.New("initial error cause must be empty")
	}
	return nil
}

// GetBid returns a bid by id.
// If a bid is not found for id, ErrBidNotFound is returned.
func (s *Store) GetBid(id broker.BidID) (*Bid, error) {
	b, err := getBid(s.store, id)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func getBid(reader ds.Read, id broker.BidID) (*Bid, error) {
	val, err := reader.Get(dsPrefix.ChildString(string(id)))
	if errors.Is(err, ds.ErrNotFound) {
		return nil, ErrBidNotFound
	} else if err != nil {
		return nil, fmt.Errorf("getting key: %v", err)
	}
	r, err := decode(val)
	if err != nil {
		return nil, fmt.Errorf("decoding value: %v", err)
	}
	return &r, nil
}

// SetAwaitingProposalCid updates bid status to BidStatusAwaitingProposal.
func (s *Store) SetAwaitingProposalCid(id broker.BidID) error {
	txn, err := s.store.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %v", err)
	}
	defer txn.Discard()

	b, err := getBid(txn, id)
	if err != nil {
		return err
	}
	if b.Status != BidStatusSubmitted {
		return fmt.Errorf("bid must have status '%s'", BidStatusSubmitted)
	}
	if err := s.saveAndTransitionStatus(txn, b, BidStatusAwaitingProposal); err != nil {
		return fmt.Errorf("updating bid: %v", err)
	}
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %v", err)
	}

	log.Debugf("awaiting bid %s proposal cid", b.ID)
	return nil
}

// SetProposalCid sets the bid proposal cid and updates status to BidStatusQueuedProposalCid.
func (s *Store) SetProposalCid(id broker.BidID, pcid cid.Cid) error {
	if !pcid.Defined() {
		return errors.New("proposal cid must be defined")
	}

	txn, err := s.store.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %v", err)
	}
	defer txn.Discard()

	b, err := getBid(txn, id)
	if err != nil {
		return err
	}
	if b.Status != BidStatusAwaitingProposal {
		return fmt.Errorf("bid must have status '%s'", BidStatusAwaitingProposal)
	}

	b.ProposalCid = pcid
	if err := s.enqueueProposalCid(txn, b); err != nil {
		return fmt.Errorf("enqueueing proposal cid: %v", err)
	}

	log.Debugf("enqueued bid %s proposal cid %s", b.ID, b.ProposalCid)
	return nil
}

// Query is used to query for bids.
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

// ListBids lists bids by applying a Query.
func (s *Store) ListBids(query Query) ([]Bid, error) {
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

	results, err := s.store.QueryExtended(dsextensions.QueryExt{
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

	var list []Bid
	for res := range results.Next() {
		if res.Error != nil {
			return nil, fmt.Errorf("getting next result: %v", res.Error)
		}
		b, err := decode(res.Value)
		if err != nil {
			return nil, fmt.Errorf("decoding value: %v", err)
		}
		list = append(list, b)
	}

	// Remove seek from list
	if len(query.Offset) != 0 && len(list) > 0 {
		list = list[1:]
	}

	return list, nil
}

// enqueueProposalCid queues a proposal cid delivery.
// commitTxn will be committed internally!
func (s *Store) enqueueProposalCid(commitTxn ds.Txn, b *Bid) error {
	// Set the bid to "fetching_proposal"
	if err := s.saveAndTransitionStatus(commitTxn, b, BidStatusFetchingProposal); err != nil {
		return fmt.Errorf("updating status (fetching_proposal): %v", err)
	}
	if commitTxn != nil {
		if err := commitTxn.Commit(); err != nil {
			return fmt.Errorf("committing txn: %v", err)
		}
	}

	// Unblock the caller by letting the rest happen in the background
	go func() {
		select {
		case s.jobCh <- b:
		default:
			log.Debugf("workers are busy; queueing %s", b.ID)
			// Workers are busy, set back to "queued"
			if err := s.saveAndTransitionStatus(nil, b, BidStatusQueuedProposal); err != nil {
				log.Errorf("updating status (queued_proposal): %v", err)
			}
		}
	}()
	return nil
}

func (s *Store) fetchWorker(num int) {
	fail := func(b *Bid, err error) (status BidStatus) {
		b.ErrorCause = err.Error()
		if b.ProposalCidFetchAttempts >= s.proposalFetchAttempts {
			status = BidStatusFinalized
			log.Warnf("job %s exhausted all %d attempts with error: %v", b.ID, s.proposalFetchAttempts, err)
		} else {
			status = BidStatusQueuedProposal
			log.Debugf("retrying job %s with error: %v", b.ID, err)
		}
		return status
	}

	for {
		select {
		case <-s.ctx.Done():
			return

		case b := <-s.jobCh:
			if s.ctx.Err() != nil {
				return
			}
			b.ProposalCidFetchAttempts++
			log.Debugf(
				"worker %d got job %s (attempt=%d/%d)", num, b.ID, b.ProposalCidFetchAttempts, s.proposalFetchAttempts)

			// Handle the auction with the runner func
			var status BidStatus
			ctx, cancel := context.WithTimeout(s.ctx, ProposalFetchTimeout)
			if err := s.writeProposalData(ctx, b.ProposalCid); err != nil {
				status = fail(b, err)
			} else {
				status = BidStatusFinalized
				// Reset error
				b.ErrorCause = ""
			}

			// Save and update status to "ended" or "error"
			if err := s.saveAndTransitionStatus(nil, b, status); err != nil {
				log.Errorf("updating status (%s): %v", status, err)
			}

			log.Debugf("worker %d finished job %s", num, b.ID)
			cancel()
			select {
			case s.tickCh <- struct{}{}:
			default:
			}
		}
	}
}

func (s *Store) writeProposalData(ctx context.Context, pcid cid.Cid) error {
	f, err := os.Create(filepath.Join(s.proposalDataDirectory, pcid.String()))
	if err != nil {
		return fmt.Errorf("opening file for proposal data: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("closing data file: %v", err)
		}
	}()

	if err := car.WriteCar(ctx, s.nodeGetter, []cid.Cid{pcid}, f); err != nil {
		return fmt.Errorf("fetching proposal cid %s: %v", pcid, err)
	}
	return nil
}

func (s *Store) startFetching() {
	t := time.NewTimer(ProposalFetchStartDelay)
	for {
		select {
		case <-s.ctx.Done():
			t.Stop()
			return
		case <-t.C:
			s.getNext()
		case <-s.tickCh:
			s.getNext()
		}
	}
}

func (s *Store) getNext() {
	txn, err := s.store.NewTransaction(false)
	if err != nil {
		log.Errorf("creating txn: %v", err)
		return
	}
	defer txn.Discard()

	b, err := s.getQueued(txn)
	if err != nil {
		log.Errorf("getting next in queue: %v", err)
		return
	}
	if b == nil {
		return
	}
	log.Debugf("enqueueing job: %s", b.ID)
	if err := s.enqueueProposalCid(txn, b); err != nil {
		log.Errorf("enqueueing: %v", err)
	}
}

func (s *Store) getQueued(txn ds.Txn) (*Bid, error) {
	results, err := txn.Query(dsq.Query{
		Prefix:   dsQueuePrefix.String(),
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

	b, err := getBid(txn, broker.BidID(path.Base(res.Key)))
	if err != nil {
		return nil, fmt.Errorf("getting bid: %v", err)
	}
	return b, nil
}

func (s *Store) getOrphaned() error {
	txn, err := s.store.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %v", err)
	}
	defer txn.Discard()

	bids, err := s.getStarted(txn)
	if err != nil {
		return fmt.Errorf("getting next in queue: %v", err)
	}
	if len(bids) == 0 {
		return nil
	}

	for _, b := range bids {
		log.Debugf("enqueueing orphaned job: %s", b.ID)
		if err := s.enqueueProposalCid(txn, &b); err != nil {
			return fmt.Errorf("enqueueing: %v", err)
		}
	}
	return nil
}

func (s *Store) getStarted(txn ds.Txn) ([]Bid, error) {
	results, err := txn.Query(dsq.Query{
		Prefix:   dsStartedPrefix.String(),
		Orders:   []dsq.Order{dsq.OrderByKey{}},
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

	var bids []Bid
	for res := range results.Next() {
		if res.Error != nil {
			return nil, fmt.Errorf("getting next result: %v", res.Error)
		}
		b, err := getBid(txn, broker.BidID(path.Base(res.Key)))
		if err != nil {
			return nil, fmt.Errorf("getting bid: %v", err)
		}
		bids = append(bids, *b)
	}
	return bids, nil
}

// saveAndTransitionStatus saves bid state and transitions to a new status.
// Do not directly edit the bids status because it is needed to determine the correct status transition.
// Pass the desired new status with newStatus.
func (s *Store) saveAndTransitionStatus(txn ds.Txn, b *Bid, newStatus BidStatus) error {
	commitTxn := txn == nil
	if commitTxn {
		var err error
		txn, err = s.store.NewTransaction(false)
		if err != nil {
			return fmt.Errorf("creating txn: %v", err)
		}
		defer txn.Discard()
	}

	if b.Status != newStatus {
		// Handle currently "queued_proposal" and "fetching_proposal" status
		if b.Status == BidStatusQueuedProposal {
			if err := txn.Delete(dsQueuePrefix.ChildString(string(b.ID))); err != nil {
				return fmt.Errorf("deleting from queue: %v", err)
			}
		} else if b.Status == BidStatusFetchingProposal {
			if err := txn.Delete(dsStartedPrefix.ChildString(string(b.ID))); err != nil {
				return fmt.Errorf("deleting from started: %v", err)
			}
		}
		if newStatus == BidStatusQueuedProposal {
			if err := txn.Put(dsQueuePrefix.ChildString(string(b.ID)), nil); err != nil {
				return fmt.Errorf("putting to queue: %v", err)
			}
		} else if newStatus == BidStatusFetchingProposal {
			if err := txn.Put(dsStartedPrefix.ChildString(string(b.ID)), nil); err != nil {
				return fmt.Errorf("putting to started: %v", err)
			}
		}
		// Update status
		b.Status = newStatus
	}

	b.UpdatedAt = time.Now()
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = b.CreatedAt
	}

	val, err := encode(b)
	if err != nil {
		return fmt.Errorf("encoding value: %v", err)
	}
	if err := txn.Put(dsPrefix.ChildString(string(b.ID)), val); err != nil {
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

func decode(v []byte) (b Bid, err error) {
	dec := gob.NewDecoder(bytes.NewReader(v))
	if err := dec.Decode(&b); err != nil {
		return b, err
	}
	return b, nil
}
