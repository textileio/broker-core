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
	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	golog "github.com/ipfs/go-log/v2"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/broker-core/broker"
	core "github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/dshelper/txndswrap"
	dsextensions "github.com/textileio/go-datastore-extensions"
)

const (
	invalidStatus = "invalid"

	// defaultListLimit is the default list page size.
	defaultListLimit = 10
	// maxListLimit is the max list page size.
	maxListLimit = 1000
)

var (
	log = golog.Logger("bidbot/store")

	// ErrBidNotFound indicates the requested bid was not found.
	ErrBidNotFound = errors.New("bid not found")

	// dsPrefix is the prefix for bids.
	// Structure: /bids/<bid_id> -> Bid.
	dsPrefix = ds.NewKey("/bids")
)

// Bid defines the core bid model from a miner's perspective.
type Bid struct {
	ID                          core.BidID
	AuctionID                   core.AuctionID
	Status                      BidStatus
	AskPrice                    int64 // attoFIL per GiB per epoch
	VerifiedAskPrice            int64 // attoFIL per GiB per epoch
	StartEpoch                  uint64
	FastRetrieval               bool
	ProposalCid                 cid.Cid
	ProposalCidDownloadAttempts uint32
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
	ErrorCause                  string
}

// BidStatus is the status of a Bid.
type BidStatus int

const (
	// BidStatusUnspecified indicates the initial or invalid status of a bid.
	BidStatusUnspecified BidStatus = iota
	// BidStatusSubmitted indicates the bid was succesfully submitted to the auctioneer.
	BidStatusSubmitted
	// BidStatusAccepted indicates the bid was accepted and is awaiting proposal cid from auctioneer.
	BidStatusAccepted
	// BidStatusQueuedProposalCid indicates the bid proposal cid was received but downloading is queued.
	BidStatusQueuedProposalCid
	// BidStatusDownloadingProposalCid indicates the bid proposal cid is downloading.
	BidStatusDownloadingProposalCid
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
	case BidStatusAccepted:
		return "pending_proposal_cid"
	case BidStatusQueuedProposalCid:
		return "queued_proposal_cid"
	case BidStatusDownloadingProposalCid:
		return "downloading_proposal_cid"
	case BidStatusFinalized:
		return "finalized"
	default:
		return invalidStatus
	}
}

// Store persists miner bids.
type Store struct {
	store   txndswrap.TxnDatastore
	entropy *ulid.MonotonicEntropy
	lk      sync.Mutex
}

// NewStore returns a new Store.
func NewStore(store txndswrap.TxnDatastore) *Store {
	return &Store{store: store}
}

// SaveBid saves a bid that has been submitted to an auctioneer.
func (s *Store) SaveBid(bid Bid) error {
	if err := validate(bid); err != nil {
		return fmt.Errorf("invalid bid data: %s", err)
	}
	id, err := s.newID(time.Now())
	if err != nil {
		return fmt.Errorf("creating bid id: %v", err)
	}
	bid.ID = id
	bid.CreatedAt = time.Now()
	bid.Status = BidStatusSubmitted

	if err := s.save(nil, &bid); err != nil {
		return fmt.Errorf("saving bid: %v", err)
	}
	return nil
}

func validate(b Bid) error {
	if b.AuctionID == "" {
		return errors.New("auction id is empty")
	}
	if b.Status != BidStatusUnspecified {
		return errors.New("invalid initial bid status")
	}
	if b.ProposalCid.Defined() {
		return errors.New("initial proposal cid cannot be defined")
	}
	if b.ProposalCidDownloadAttempts != 0 {
		return errors.New("initial proposal cid download attempts must be zero")
	}
	if b.ErrorCause != "" {
		return errors.New("initial error cause must be empty")
	}
	return nil
}

// newID returns new monotonically increasing bid ids.
func (s *Store) newID(t time.Time) (broker.BidID, error) {
	s.lk.Lock() // entropy is not safe for concurrent use

	if s.entropy == nil {
		s.entropy = ulid.Monotonic(rand.Reader, 0)
	}
	id, err := ulid.New(ulid.Timestamp(t.UTC()), s.entropy)
	if errors.Is(err, ulid.ErrMonotonicOverflow) {
		s.entropy = nil
		s.lk.Unlock()
		return s.newID(t)
	} else if err != nil {
		s.lk.Unlock()
		return "", fmt.Errorf("generating id: %v", err)
	}
	s.lk.Unlock()
	return broker.BidID(strings.ToLower(id.String())), nil
}

// GetBid returns an bid by id.
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
		return nil, fmt.Errorf("getting bid: %v", err)
	}
	r, err := decode(val)
	if err != nil {
		return nil, fmt.Errorf("decoding bid: %v", err)
	}
	return &r, nil
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
	defer func() { _ = results.Close() }()

	var list []Bid
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

	log.Debugf("listed %d bids", len(list))
	return list, nil
}

// SetAccepted updates bid status to BidStatusAccepted.
func (s *Store) SetAccepted(id core.BidID) error {
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

	b.Status = BidStatusAccepted
	if err := s.save(txn, b); err != nil {
		return fmt.Errorf("updating bid: %v", err)
	}
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %v", err)
	}

	log.Debugf("set bid %s status to '%s'", b.ID, b.Status)
	return nil
}

// SetProposalCid sets the bid proposal cid and updates status to BidStatusQueuedProposalCid.
func (s *Store) SetProposalCid(id core.BidID, pcid cid.Cid) error {
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
	if b.Status != BidStatusAccepted {
		return fmt.Errorf("bid must have status '%s'", BidStatusAccepted)
	}

	b.ProposalCid = pcid
	b.Status = BidStatusQueuedProposalCid
	if err := s.save(txn, b); err != nil {
		return fmt.Errorf("updating bid: %v", err)
	}
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %v", err)
	}

	log.Debugf("set bid %s status to '%s'", b.ID, b.Status)
	return nil
}

// SetProposalCidDownloading updates bid status to BidStatusDownloadingProposalCid.
func (s *Store) SetProposalCidDownloading(id core.BidID) error {
	txn, err := s.store.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %v", err)
	}
	defer txn.Discard()

	b, err := getBid(txn, id)
	if err != nil {
		return err
	}
	if b.Status != BidStatusQueuedProposalCid {
		return fmt.Errorf("bid must have status '%s'", BidStatusQueuedProposalCid)
	}

	b.Status = BidStatusDownloadingProposalCid
	if err := s.save(txn, b); err != nil {
		return fmt.Errorf("updating bid: %v", err)
	}
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %v", err)
	}

	log.Debugf("set bid %s status to '%s'", b.ID, b.Status)
	return nil
}

// SetFinalized optionally sets an error cause and updates bid status to BidStatusFinalized.
func (s *Store) SetFinalized(id core.BidID) error {
	txn, err := s.store.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %v", err)
	}
	defer txn.Discard()

	b, err := getBid(txn, id)
	if err != nil {
		return err
	}
	if b.Status != BidStatusDownloadingProposalCid {
		return fmt.Errorf("bid must have status '%s'", BidStatusDownloadingProposalCid)
	}

	b.Status = BidStatusFinalized
	if err := s.save(txn, b); err != nil {
		return fmt.Errorf("updating bid: %v", err)
	}
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %v", err)
	}

	log.Debugf("set bid %s status to '%s'", b.ID, b.Status)
	return nil
}

func (s *Store) save(txn ds.Txn, b *Bid) error {
	var writer ds.Write
	if txn != nil {
		writer = txn
	} else {
		writer = s.store
	}

	b.UpdatedAt = time.Now()
	val, err := encode(b)
	if err != nil {
		return fmt.Errorf("encoding bid: %v", err)
	}
	if err := writer.Put(dsPrefix.ChildString(string(b.ID)), val); err != nil {
		return fmt.Errorf("putting bid: %v", err)
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
	var buf bytes.Buffer
	if _, err := buf.Write(v); err != nil {
		return b, err
	}
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&b); err != nil {
		return b, err
	}
	return b, nil
}
