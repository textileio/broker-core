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
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/dshelper/txndswrap"
)

var (
	// ErrNotFound is returned if the item isn't found in the store.
	ErrNotFound = errors.New("key isn't found")

	dsPrefixAuctionData = datastore.NewKey("/auction-data")
	dsPrefixAuctionDeal = datastore.NewKey("/auction-deal")

	log = logger.Logger("dealer/store")
)

// AuctionData contains information of data to be stored in Filecoin.
type AuctionData struct {
	ID string

	StorageDealID broker.StorageDealID
	PayloadCid    cid.Cid
	PieceCid      cid.Cid
	PieceSize     uint64
	Duration      uint64

	CreatedAt time.Time
	UpdatedAt time.Time
}

// AuctionDealStatus is the type of action deal status.
type AuctionDealStatus int

const (
	// Pending indicates that an auction data is ready for deal making.
	Pending AuctionDealStatus = iota
	// WaitingConfirmation indicates that an auction data deal should be monitored until is active on-chain.
	WaitingConfirmation
	// Error is a final status that indicates the deal process failed. More details about the error
	// are found in the ErorCause field.
	Error
	// Success is a final status tha indicates the deal process succeeded.
	Success
)

// AuctionDeal contains information to make a deal with a particular miner. The data information is stored
// in the linked AuctionData.
type AuctionDeal struct {
	ID string

	AuctionDataID       string
	Miner               string
	PricePerGiBPerEpoch int64
	StartEpoch          uint64
	Verified            bool
	FastRetrieval       bool

	Status     AuctionDealStatus
	ErrorCause string

	CreatedAt time.Time
	UpdatedAt time.Time

	ProposalCid    cid.Cid
	DealID         int64
	DealExpiration uint64
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
func New(ds txndswrap.TxnDatastore) (*Store, error) {
	s := &Store{
		ds:          ds,
		auctionData: map[string]AuctionData{},
	}
	if err := s.loadCache(); err != nil {
		return nil, fmt.Errorf("loading in-memory cache: %s", err)
	}

	return s, nil
}

// Create persist new auction data and a set of related auction deals.
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
	ad.CreatedAt = time.Now()
	if err := s.save(txn, makeAuctionDataKey(ad.ID), ad); err != nil {
		return fmt.Errorf("saving auction data in datastore: %s", err)
	}

	// Save all AuctionDeals linked to this AuctionData.
	for i := range ads {
		newID, err := s.newID()
		if err != nil {
			return fmt.Errorf("generating new id: %s", err)
		}
		ads[i].ID = newID
		ads[i].AuctionDataID = ad.ID // Link with its AuctionData.
		ads[i].Status = Pending
		ads[i].CreatedAt = ad.CreatedAt
		if err := s.save(txn, makeAuctionDealKey(ads[i].ID), ads[i]); err != nil {
			return fmt.Errorf("saving auction deal in datastore: %s", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	// Include them in our in-memory cache.
	s.lock.Lock()
	s.auctionData[ad.ID] = ad
	s.auctionDeals = append(s.auctionDeals, ads...)
	// Re-sorting since the lock wasn't acquired at the start
	// of the func for performance reasons, so many calls can race
	// at different times concurrently.
	sort.Slice(s.auctionDeals, func(i, j int) bool {
		return s.auctionDeals[i].ID < s.auctionDeals[j].ID
	})
	s.lock.Unlock()

	return nil
}

// SaveAuctionDeal persists a modified auction deal.
func (s *Store) SaveAuctionDeal(aud AuctionDeal) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	cacheIdx := -1
	for i := range s.auctionDeals {
		if s.auctionDeals[i].ID == aud.ID {
			cacheIdx = i
			break
		}
	}
	if cacheIdx == -1 {
		return ErrNotFound
	}
	currentAud := s.auctionDeals[cacheIdx]

	if currentAud.Status != aud.Status {
		if err := isValidStatusChange(currentAud.Status, aud); err != nil {
			return fmt.Errorf("invalid status change: %s", err)
		}
	}

	aud.UpdatedAt = time.Now()
	if err := s.save(s.ds, makeAuctionDealKey(aud.ID), aud); err != nil {
		return fmt.Errorf("saving auction deal status change: %s", err)
	}

	// After we're sure was updated in the datastore, update it in the cache.
	s.auctionDeals[cacheIdx] = aud

	return nil
}

// GetAllAuctionDeals returns all auction deals with a particular status.
func (s *Store) GetAllAuctionDeals(status AuctionDealStatus) ([]AuctionDeal, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	var res []AuctionDeal
	for _, aud := range s.auctionDeals {
		if aud.Status != status {
			continue
		}
		res = append(res, aud)
	}

	return res, nil
}

// GetAuctionData returns an auction data by id. If not found returns ErrNotFound.
func (s *Store) GetAuctionData(auctionDataID string) (AuctionData, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ad, ok := s.auctionData[auctionDataID]
	if !ok {
		return AuctionData{}, ErrNotFound
	}
	return ad, nil
}

// RemoveAuctionDeals removes the provided auction deals. If the corresponding auction data isn't linked
// with any remaining auction deal, then those are also removed.
func (s *Store) RemoveAuctionDeals(ads []AuctionDeal) error {
	for _, aud := range ads {
		if aud.Status != Error && aud.Status != Success {
			return fmt.Errorf("only auction deals in final status can be removed")
		}
		if err := s.ds.Delete(makeAuctionDealKey(aud.ID)); err != nil {
			return fmt.Errorf("deleting auction deal: %s", err)
		}
		s.lock.Lock()
		for i := range s.auctionDeals {
			if s.auctionDeals[i].ID == aud.ID {
				s.auctionDeals = append(s.auctionDeals[:i], s.auctionDeals[i+1:]...)
				break
			}
		}
		s.lock.Unlock()
	}

	// Investigate AuctionData that are still alive, so we can GC orpahaned
	// AuctionData that we can also delete.
	aliveADs := map[string]struct{}{}
	s.lock.Lock()
	for _, aud := range s.auctionDeals {
		aliveADs[aud.AuctionDataID] = struct{}{}
	}
	var deletableADs []string
	for adID := range s.auctionData {
		if _, ok := aliveADs[adID]; !ok {
			deletableADs = append(deletableADs, adID)
		}
	}
	s.lock.Unlock()

	for _, id := range deletableADs {
		if err := s.ds.Delete(makeAuctionDataKey(id)); err != nil {
			return fmt.Errorf("deleting auction data: %s", err)
		}
		s.lock.Lock()
		delete(s.auctionData, id)
		s.lock.Unlock()
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
	if ad.StorageDealID == "" {
		return errors.New("storage deal id is empty")
	}
	if !ad.PayloadCid.Defined() {
		return errors.New("payload cid is undefined")
	}
	if !ad.PieceCid.Defined() {
		return errors.New("piece cid is undefined")
	}
	if ad.PieceSize <= 0 {
		return errors.New("piece size is zero")
	}

	for _, auctionDeal := range ads {
		if auctionDeal.Miner == "" {
			return errors.New("miner address is empty")
		}
		if auctionDeal.PricePerGiBPerEpoch < 0 {
			return errors.New("price-per-gib-per-epoch is negative")
		}
		if auctionDeal.StartEpoch <= 0 {
			return errors.New("start-epoch isn't positive")
		}
	}

	return nil
}

func isValidStatusChange(pre AuctionDealStatus, aud AuctionDeal) error {
	switch pre {
	case Pending:
		if aud.Status != WaitingConfirmation {
			return fmt.Errorf("expecting WaitingConfirmation but found: %s", aud.Status)
		}
		if !aud.ProposalCid.Defined() {
			return errors.New("proposal cid should be set to transition to WaitingConfirmation")
		}
	case WaitingConfirmation:
		if aud.Status != Error && aud.Status != Success {
			return fmt.Errorf("expecting final status but found: %s", aud.Status)
		}
		if aud.Status == Error && aud.ErrorCause == "" {
			return errors.New("an error status should have an error cause")
		}
		if aud.Status == Success && aud.ErrorCause != "" {
			return fmt.Errorf("a success status can't have an error cause: %s", aud.ErrorCause)
		}
	case Success:
	case Error:
		return fmt.Errorf("error/success status are final, so %s isn't allowed", aud.Status)
	default:
		return fmt.Errorf("unknown status: %s", aud.Status)
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
