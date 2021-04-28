package store

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	logger "github.com/ipfs/go-log/v2"
	"github.com/oklog/ulid/v2"
)

var (
	ErrNotFound = fmt.Errorf("key isn't found")

	dsPrefixAuctionData = datastore.NewKey("/auction-data")
	dsPrefixAuctionDeal = datastore.NewKey("/auction-deal")

	log = logger.Logger("dealer/store")
)

type AuctionData struct {
	ID string

	AuctionID  string
	PayloadCid cid.Cid
	PieceCid   cid.Cid
	PieceSize  int64
	Duration   int64
}

type AuctionDealStatus int

const (
	Pending AuctionDealStatus = iota
	WaitingConfirmation
	Error
	Success
)

type AuctionDeal struct {
	ID string

	AuctionDataID       string
	Miner               string
	PricePerGiBPerEpoch int64
	StartEpoch          int64
	Verified            bool
	FastRetrieval       bool

	Status     AuctionDealStatus
	ErrorCause string
}

// Store provides persistent storage for Bids.
type Store struct {
	ds      datastore.TxnDatastore
	entropy *ulid.MonotonicEntropy

	lock         sync.Mutex
	auctionData  map[string]AuctionData
	auctionDeals []AuctionDeal
}

// New returns a *Store.
func New(ds datastore.TxnDatastore) (*Store, error) {
	s := &Store{
		ds:          ds,
		auctionData: map[string]AuctionData{},
	}
	if err := s.loadCache(); err != nil {
		return nil, fmt.Errorf("loading in-memory cache: %s", err)
	}

	return s, nil
}

func (s *Store) Create(ad AuctionData, ads []AuctionDeal) error {
	if err := validate(ad, ads); err != nil {
		return fmt.Errorf("invalid auction data: %s", err)
	}

	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %s", err)
	}
	defer txn.Discard()

	// Save AuctionData.
	newID, err := s.newID()
	if err != nil {
		return fmt.Errorf("generating new id: %s", err)
	}
	ad.ID = newID
	if err := s.save(txn, makeAuctionDataKey(ad.ID), ad); err != nil {
		return fmt.Errorf("saving auction data in datastore: %s", err)
	}

	// Save all AuctionDeals linked to this AuctionData.
	for _, auctionDeal := range ads {
		newID, err := s.newID()
		if err != nil {
			return fmt.Errorf("generating new id: %s", err)
		}
		auctionDeal.ID = newID
		auctionDeal.AuctionDataID = ad.ID // Link with its AuctionData.
		auctionDeal.Status = Pending
		if err := s.save(txn, makeAuctionDealKey(auctionDeal.ID), auctionDeal); err != nil {
			return fmt.Errorf("saving auction deal in datastore: %s", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	// Include them in our in-memory cache.
	s.lock.Lock()
	s.auctionData[ad.ID] = ad
	for _, auctionDeal := range ads {
		s.auctionDeals = append(s.auctionDeals, auctionDeal)
	}
	// Re-sorting since the lock wasn't adquired at the start
	// of the func for performance reasons, so many calls can race
	// at different times concurrently.
	sort.Slice(s.auctionDeals, func(i, j int) bool {
		return s.auctionDeals[i].ID < s.auctionDeals[j].ID
	})
	s.lock.Unlock()

	return nil
}

func (s *Store) StatusChange(auctionDealID string, newStatus AuctionDealStatus) error {
	s.lock.Lock()
	var ad *AuctionDeal
	for i := range s.auctionDeals {
		if s.auctionDeals[i].ID == auctionDealID {
			ad = &s.auctionDeals[i]
		}
	}
	s.lock.Unlock()
	if ad == nil {
		return ErrNotFound
	}

	if err := isValidStatusChange(ad.Status, newStatus); err != nil {
		return fmt.Errorf("invalid status change: %s", err)
	}

	// Change auction-deal status, and at the same time update it in the cache,
	// since `ad` is a pointer to the corresponding item.
	s.lock.Lock()
	ad.Status = newStatus
	s.lock.Unlock()

	if err := s.save(s.ds, makeAuctionDealKey(ad.ID), ad); err != nil {
		return fmt.Errorf("saving auction deal status change: %s", err)
	}

	return nil
}

func (s *Store) save(dsWrite datastore.Write, id datastore.Key, b interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(b); err != nil {
		return fmt.Errorf("gob encoding: %s", err)
	}
	if err := dsWrite.Put(id, buf.Bytes()); err != nil {
		return fmt.Errorf("put in datastore: %s", err)
	}

	return nil
}

func (s *Store) newID() (string, error) {
	s.lock.Lock()
	// Not deferring unlock since can be recursive.

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

func (s *Store) loadCache() error {
	// AuctionData
	log.Debugf("loading auction data")
	q := query.Query{
		Prefix: dsPrefixAuctionData.String(),
		Orders: []query.Order{query.OrderByKey{}},
	}
	res, err := s.ds.Query(q)
	if err != nil {
		return fmt.Errorf("creating query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing query result: %s", err)
		}
	}()

	for item := range res.Next() {
		if item.Error != nil {
			return fmt.Errorf("fetching auction data item result: %s", item.Error)
		}
		var ad AuctionData
		d := gob.NewDecoder(bytes.NewReader(item.Value))
		if err := d.Decode(&ad); err != nil {
			return fmt.Errorf("unmarshaling gob: %s", err)
		}
		s.auctionData[item.Key] = ad
	}

	// AuctionDeals
	log.Debugf("loading auction deals")
	q = query.Query{
		Prefix: dsPrefixAuctionDeal.String(),
		Orders: []query.Order{query.OrderByKey{}},
	}
	res, err = s.ds.Query(q)
	if err != nil {
		return fmt.Errorf("creating query: %s", err)
	}
	defer func() {
		if err := res.Close(); err != nil {
			log.Errorf("closing query result: %s", err)
		}
	}()

	for item := range res.Next() {
		if item.Error != nil {
			return fmt.Errorf("fetching auction deal item result: %s", item.Error)
		}
		var ad AuctionDeal
		d := gob.NewDecoder(bytes.NewReader(item.Value))
		if err := d.Decode(&ad); err != nil {
			return fmt.Errorf("unmarshaling gob: %s", err)
		}
		s.auctionDeals = append(s.auctionDeals, ad)
	}

	return nil
}

func validate(ad AuctionData, ads []AuctionDeal) error {
	if ad.Duration <= 0 {
		return fmt.Errorf("invalid duration: %d", ad.Duration)
	}
	if ad.AuctionID == "" {
		return fmt.Errorf("auction-id is empty")
	}
	if !ad.PayloadCid.Defined() {
		return fmt.Errorf("payload-cid is undefined")
	}
	if !ad.PieceCid.Defined() {
		return fmt.Errorf("piece-cid is undefined")
	}
	if ad.PieceSize <= 0 {
		return fmt.Errorf("piece-size is zero")
	}

	for _, auctionDeal := range ads {
		if auctionDeal.Miner == "" {
			return fmt.Errorf("miner address is empty")
		}
		if auctionDeal.PricePerGiBPerEpoch < 0 {
			return fmt.Errorf("price-per-gib-per-epoch is negative")
		}
		if auctionDeal.StartEpoch <= 0 {
			return fmt.Errorf("start-epoch isn't positive")
		}
	}

	return nil
}

func isValidStatusChange(pre AuctionDealStatus, post AuctionDealStatus) error {
	switch pre {
	case Pending:
		if post != WaitingConfirmation {
			return fmt.Errorf("expecting WaitingConfirmation but found: %s", post)
		}
	case WaitingConfirmation:
		if post != Error && post != Success {
			return fmt.Errorf("expecting final status but found: %s", post)
		}
	case Success:
	case Error:
		return fmt.Errorf("error/success status are final, so %s isn't allowed", post)
	default:
		return fmt.Errorf("unknown status: %s", pre)
	}

	return nil
}

func makeAuctionDataKey(id string) datastore.Key {
	return dsPrefixAuctionData.ChildString(id)
}

func makeAuctionDealKey(id string) datastore.Key {
	return dsPrefixAuctionDeal.ChildString(id)
}

func (ads AuctionDealStatus) String() string {
	switch ads {
	case Pending:
		return "Pending"
	case WaitingConfirmation:
		return "WaitingConfirmation"
	case Success:
		return "Success"
	case Error:
		return "Error"
	default:
		panic("unknown deal status")
	}
}
