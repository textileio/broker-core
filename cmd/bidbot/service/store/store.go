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
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	ipfsconfig "github.com/ipfs/go-ipfs-config"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipld/go-car"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/dshelper/txndswrap"
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
	log = golog.Logger("bidbot/store")

	// DataCidFetchStartDelay is the time delay before the store will process queued data cid fetches on start.
	DataCidFetchStartDelay = time.Second * 10

	// DataCidFetchTimeout is the timeout used when fetching data cids.
	DataCidFetchTimeout = time.Hour

	// MaxDataCidFetchConcurrency is the maximum number of data cid fetches that will be handled concurrently.
	MaxDataCidFetchConcurrency = 10

	// ErrBidNotFound indicates the requested bid was not found.
	ErrBidNotFound = errors.New("bid not found")

	// dsPrefix is the prefix for bids.
	// Structure: /bids/<bid_id> -> Bid.
	dsPrefix = ds.NewKey("/bids")

	// dsQueuedPrefix is the prefix for queued data cid fetches.
	// Structure: /data_queued/<bid_id> -> nil.
	dsQueuedPrefix = ds.NewKey("/data_queued")

	// dsFetchingPrefix is the prefix for fetching data cid fetches.
	// Structure: /data_fetching/<bid_id> -> nil.
	dsFetchingPrefix = ds.NewKey("/data_fetching")
)

// Bid defines the core bid model from a miner's perspective.
type Bid struct {
	ID                   broker.BidID
	AuctionID            broker.AuctionID
	AuctioneerID         peer.ID
	DataCid              cid.Cid
	DealSize             uint64
	DealDuration         uint64
	Status               BidStatus
	AskPrice             int64 // attoFIL per GiB per epoch
	VerifiedAskPrice     int64 // attoFIL per GiB per epoch
	StartEpoch           uint64
	FastRetrieval        bool
	ProposalCid          cid.Cid
	DataCidFetchAttempts uint32
	CreatedAt            time.Time
	UpdatedAt            time.Time
	ErrorCause           string
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
	// BidStatusQueuedData indicates the bid proposal cid was received but data downloading is queued.
	BidStatusQueuedData
	// BidStatusFetchingData indicates the bid data cid is being fetched.
	BidStatusFetchingData
	// BidStatusFinalized indicates the bid has reached a final state.
	// If ErrorCause is empty, the bid has been accepted and the data cid downloaded.
	// If ErrorCause is not empty, a fatal error has occurred and the bid should be considered abandoned.
	BidStatusFinalized
)

var bidStatusStrings map[BidStatus]string = map[BidStatus]string{
	BidStatusUnspecified:      "unspecified",
	BidStatusSubmitted:        "submitted",
	BidStatusAwaitingProposal: "awaiting_proposal",
	BidStatusQueuedData:       "queued_data",
	BidStatusFetchingData:     "fetching_data",
	BidStatusFinalized:        "finalized",
}

var bidStatusByString map[string]BidStatus

func init() {
	bidStatusByString = make(map[string]BidStatus)
	for b, s := range bidStatusStrings {
		bidStatusByString[s] = b
	}
}

// String returns a string-encoded status.
func (as BidStatus) String() string {
	if s, exists := bidStatusStrings[as]; exists {
		return s
	} else {
		return "invalid"
	}
}

// BidStatusByString finds a status by its string representation, or errors if
// the status does not exist.
func BidStatusByString(s string) (BidStatus, error) {
	if bs, exists := bidStatusByString[s]; exists {
		return bs, nil
	}
	return -1, errors.New("invalid bid status")
}

// Store stores miner auction deal bids.
type Store struct {
	store      txndswrap.TxnDatastore
	host       host.Host
	nodeGetter format.NodeGetter
	bootstrap  []peer.AddrInfo

	jobCh  chan *Bid
	tickCh chan struct{}

	dealDataDirectory     string
	dealDataFetchAttempts uint32

	ctx    context.Context
	cancel context.CancelFunc

	wg sync.WaitGroup
}

// NewStore returns a new Store.
func NewStore(
	store txndswrap.TxnDatastore,
	host host.Host,
	nodeGetter format.NodeGetter,
	bootstrap []string,
	dealDataDirectory string,
	dealDataFetchAttempts uint32,
) (*Store, error) {
	baddrs, err := ipfsconfig.ParseBootstrapPeers(bootstrap)
	if err != nil {
		return nil, fmt.Errorf("parsing bootstrap addrs: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &Store{
		store:                 store,
		host:                  host,
		nodeGetter:            nodeGetter,
		bootstrap:             baddrs,
		jobCh:                 make(chan *Bid, MaxDataCidFetchConcurrency),
		tickCh:                make(chan struct{}, MaxDataCidFetchConcurrency),
		dealDataDirectory:     dealDataDirectory,
		dealDataFetchAttempts: dealDataFetchAttempts,
		ctx:                   ctx,
		cancel:                cancel,
	}

	// Create data fetch workers
	s.wg.Add(MaxDataCidFetchConcurrency)
	for i := 0; i < MaxDataCidFetchConcurrency; i++ {
		go s.fetchWorker(i + 1)
	}

	// Re-enqueue jobs that may have been orphaned during an forced shutdown
	if err := s.getOrphaned(); err != nil {
		return nil, fmt.Errorf("getting orphaned jobs: %v", err)
	}

	go s.startFetching()
	return s, nil
}

// Close the store. This will wait for "fetching" data cid fetches.
func (s *Store) Close() error {
	s.cancel()
	s.wg.Wait()
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
	log.Infof("saved bid %s", bid.ID)
	return nil
}

func validate(b Bid) error {
	if b.ID == "" {
		return errors.New("id is empty")
	}
	if b.AuctionID == "" {
		return errors.New("auction id is empty")
	}
	if err := b.AuctioneerID.Validate(); err != nil {
		return fmt.Errorf("auctioneer id is not a valid peer id: %v", err)
	}
	if !b.DataCid.Defined() {
		return errors.New("data cid is not defined")
	}
	if b.DealSize == 0 {
		return errors.New("deal size must be greater than zero")
	}
	if b.DealDuration == 0 {
		return errors.New("deal duration must be greater than zero")
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
	if b.DataCidFetchAttempts != 0 {
		return errors.New("initial data cid download attempts must be zero")
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
	return r, nil
}

// SetAwaitingProposalCid updates bid status to BidStatusAwaitingProposal.
// If a bid is not found for id, ErrBidNotFound is returned.
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

	log.Infof("set awaiting proposal cid for bid %s", b.ID)
	return nil
}

// SetProposalCid sets the bid proposal cid and updates status to BidStatusQueuedData.
// If a bid is not found for id, ErrBidNotFound is returned.
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
	if err := s.enqueueDataCid(txn, b); err != nil {
		return fmt.Errorf("enqueueing data cid: %v", err)
	}

	log.Infof("set proposal cid for bid %s; enqueued data cid %s for download", b.ID, b.DataCid)
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
func (s *Store) ListBids(query Query) ([]*Bid, error) {
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

	var list []*Bid
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

// WriteCar writes a car file to the configured deal data directory.
func (s *Store) WriteCar(ctx context.Context, pcid cid.Cid) (string, error) {
	f, err := os.Create(filepath.Join(s.dealDataDirectory, pcid.String()))
	if err != nil {
		return "", fmt.Errorf("opening file for deal data: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("closing data file: %v", err)
		}
	}()

	for _, dial := range s.bootstrap {
		go func(dial peer.AddrInfo) {
			if err := s.host.Connect(ctx, dial); err != nil {
				log.Errorf("dialing %s: %v", dial.ID, err)
			}
		}(dial)
	}

	if err := car.WriteCar(ctx, s.nodeGetter, []cid.Cid{pcid}, f); err != nil {
		return "", fmt.Errorf("fetching data cid %s: %v", pcid, err)
	}
	return f.Name(), nil
}

// enqueueDataCid queues a data cid fetch.
// commitTxn will be committed internally!
func (s *Store) enqueueDataCid(commitTxn ds.Txn, b *Bid) error {
	// Set the bid to "fetching_data"
	if err := s.saveAndTransitionStatus(commitTxn, b, BidStatusFetchingData); err != nil {
		return fmt.Errorf("updating status (fetching_data): %v", err)
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
			// Workers are busy, set back to "queued_data"
			if err := s.saveAndTransitionStatus(nil, b, BidStatusQueuedData); err != nil {
				log.Errorf("updating status (queued_data): %v", err)
			}
		}
	}()
	return nil
}

func (s *Store) fetchWorker(num int) {
	defer s.wg.Done()

	fail := func(b *Bid, err error) (status BidStatus) {
		b.ErrorCause = err.Error()
		if b.DataCidFetchAttempts >= s.dealDataFetchAttempts {
			status = BidStatusFinalized
			log.Warnf("job %s exhausted all %d attempts with error: %v", b.ID, s.dealDataFetchAttempts, err)
		} else {
			status = BidStatusQueuedData
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
			log.Infof("downloading data cid %s", b.DataCid)
			b.DataCidFetchAttempts++
			log.Debugf(
				"worker %d got job %s (attempt=%d/%d)", num, b.ID, b.DataCidFetchAttempts, s.dealDataFetchAttempts)

			// Fetch the data cid
			var status BidStatus
			var logMsg string
			ctx, cancel := context.WithTimeout(s.ctx, DataCidFetchTimeout)
			if _, err := s.WriteCar(ctx, b.DataCid); err != nil {
				status = fail(b, err)
				logMsg = fmt.Sprintf("status=%s error=%s", status, b.ErrorCause)
			} else {
				status = BidStatusFinalized
				// Reset error
				b.ErrorCause = ""
				logMsg = fmt.Sprintf("status=%s", status)
			}
			log.Infof("finished downloading data cid %s (%s)", b.ID, logMsg)

			// Save and update status to "finalized"
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

func (s *Store) startFetching() {
	t := time.NewTimer(DataCidFetchStartDelay)
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
		log.Errorf("getting next in queued: %v", err)
		return
	}
	if b == nil {
		return
	}
	log.Debugf("enqueueing job: %s", b.ID)
	if err := s.enqueueDataCid(txn, b); err != nil {
		log.Errorf("enqueueing: %v", err)
	}
}

func (s *Store) getQueued(txn ds.Txn) (*Bid, error) {
	results, err := txn.Query(dsq.Query{
		Prefix:   dsQueuedPrefix.String(),
		Orders:   []dsq.Order{dsq.OrderByKey{}},
		Limit:    1,
		KeysOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("querying queued: %v", err)
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

	bids, err := s.getFetching(txn)
	if err != nil {
		return fmt.Errorf("getting next in queued: %v", err)
	}
	if len(bids) == 0 {
		return nil
	}

	for _, b := range bids {
		log.Debugf("enqueueing orphaned job: %s", b.ID)
		if err := s.enqueueDataCid(txn, &b); err != nil {
			return fmt.Errorf("enqueueing: %v", err)
		}
	}
	return nil
}

func (s *Store) getFetching(txn ds.Txn) ([]Bid, error) {
	results, err := txn.Query(dsq.Query{
		Prefix:   dsFetchingPrefix.String(),
		Orders:   []dsq.Order{dsq.OrderByKey{}},
		KeysOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("querying queued: %v", err)
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
		// Handle currently "queued_data" and "fetching_data" status
		if b.Status == BidStatusQueuedData {
			if err := txn.Delete(dsQueuedPrefix.ChildString(string(b.ID))); err != nil {
				return fmt.Errorf("deleting from queued: %v", err)
			}
		} else if b.Status == BidStatusFetchingData {
			if err := txn.Delete(dsFetchingPrefix.ChildString(string(b.ID))); err != nil {
				return fmt.Errorf("deleting from fetching: %v", err)
			}
		}
		if newStatus == BidStatusQueuedData {
			if err := txn.Put(dsQueuedPrefix.ChildString(string(b.ID)), nil); err != nil {
				return fmt.Errorf("putting to queued: %v", err)
			}
		} else if newStatus == BidStatusFetchingData {
			if err := txn.Put(dsFetchingPrefix.ChildString(string(b.ID)), nil); err != nil {
				return fmt.Errorf("putting to fetching: %v", err)
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

func decode(v []byte) (b *Bid, err error) {
	dec := gob.NewDecoder(bytes.NewReader(v))
	if err := dec.Decode(&b); err != nil {
		return b, err
	}
	return b, nil
}
