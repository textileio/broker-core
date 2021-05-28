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
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	logger "github.com/ipfs/go-log/v2"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/dshelper/txndswrap"
)

var (
	// ErrNotFound is returned if the item isn't found in the store.
	ErrNotFound = errors.New("key not found")

	dsPrefixAuctionData                      = datastore.NewKey("/auction-data")
	dsPrefixAuctionDealPendingDealMaking     = datastore.NewKey("/auction-deal/pending-deal-making")
	dsPrefixAuctionDealExecutingDealMaking   = datastore.NewKey("/auction-deal/executing-deal-making")
	dsPrefixAuctionDealPendingConfirmation   = datastore.NewKey("/auction-deal/pending-confirmation")
	dsPrefixAuctionDealExecutingConfirmation = datastore.NewKey("/auction-deal/executing-confirmation")
	dsPrefixAuctionPendingReportFinalized    = datastore.NewKey("/auction-deal/pending-finalized")
	dsPrefixAuctionExecutingReportFinalized  = datastore.NewKey("/auction-deal/executing-finalized")

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

	LinkedAuctionDeals int
}

// AuctionDealStatus is the type of action deal status.
type AuctionDealStatus int

const (
	// PendingDealMaking indicates that an auction data is ready for deal making.
	PendingDealMaking AuctionDealStatus = iota
	// Executing indicates that a deal is being executed.
	ExecutingDealMaking
	// WaitingConfirmation indicates that a deal is pending to be confirmed.
	PendingConfirmation
	// ExecutingConfirmation indicates that a deal confirmation is being evaluated.
	ExecutingConfirmation
	// PendingReportFinalized indicates that a deal result is pending to be reported.
	PendingReportFinalized
	// ExecutingReportFinalized indicates that a deal result is being reported.
	ExecutingReportFinalized
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
	Retries    int

	CreatedAt time.Time
	UpdatedAt time.Time
	ReadyAt   time.Time

	ProposalCid    cid.Cid
	DealID         int64
	DealExpiration uint64
}

// Store provides persistent storage for Bids.
type Store struct {
	ds datastore.TxnDatastore

	lock    sync.Mutex
	entropy *ulid.MonotonicEntropy
}

// New returns a *Store.
func New(ds txndswrap.TxnDatastore) (*Store, error) {
	s := &Store{
		ds: ds,
	}
	return s, nil
}

// Create persist new auction data and a set of related auction deals.
func (s *Store) Create(ad *AuctionData, ads []*AuctionDeal) error {
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
	ad.LinkedAuctionDeals = len(ads)
	if err := save(txn, makeAuctionDataKey(ad.ID), ad); err != nil {
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
		ads[i].Status = PendingDealMaking
		ads[i].CreatedAt = ad.CreatedAt
		ads[i].ReadyAt = time.Unix(0, 0)

		key, _ := makeAuctionDealKey(ads[i].ID, PendingDealMaking)
		if err := save(txn, key, ads[i]); err != nil {
			return fmt.Errorf("saving auction deal in datastore: %s", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

func (s *Store) GetNext(status AuctionDealStatus) (AuctionDeal, bool, error) {
	dsPrefix, err := getAuctionDealDSPrefix(status)
	if err != nil {
		return AuctionDeal{}, false, fmt.Errorf("get auction deal datastore prefix: %s", err)
	}

	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return AuctionDeal{}, false, fmt.Errorf("creating txn: %s", err)
	}
	defer txn.Discard()

	// 1. Get the first available result for the status.
	q := query.Query{Prefix: dsPrefix.String()}
	res, err := txn.Query(q)
	if err != nil {
		return AuctionDeal{}, false, fmt.Errorf("creating query: %s", err)
	}
	defer res.Close()

	now := time.Now()
	var found bool
	var ad AuctionDeal
	var oldKey datastore.Key
	for item := range res.Next() {
		if item.Error != nil {
			return AuctionDeal{}, false, fmt.Errorf("get query item: %s", item.Error)
		}
		d := gob.NewDecoder(bytes.NewReader(item.Value))
		if err := d.Decode(&ad); err != nil {
			return AuctionDeal{}, false, fmt.Errorf("unmarshaling gob: %s", err)
		}

		// This must never happen since each status
		// lives in the queried datastore namespace key.
		// Do the paranoid check.
		if ad.Status != status {
			return AuctionDeal{}, false, fmt.Errorf("status doesn't match got %s expected %s", ad.Status, status)
		}

		if ad.ReadyAt.Before(now) {
			oldKey = datastore.NewKey(item.Key)
			found = true
			break
		}
	}
	if !found {
		return AuctionDeal{}, false, nil
	}

	// 2. To "lock" the item, we move to the next status (PendingXXX -> ExecutingXXX). Delete it from the current key, and insert it to the new one.

	if err := txn.Delete(oldKey); err != nil {
		return AuctionDeal{}, false, fmt.Errorf("deleting from current key: %s", err)
	}

	switch ad.Status {
	case PendingDealMaking:
		ad.Status = ExecutingDealMaking
	case PendingConfirmation:
		ad.Status = ExecutingConfirmation
	case PendingReportFinalized:
		ad.Status = ExecutingReportFinalized
	default:
		return AuctionDeal{}, false, fmt.Errorf("invalid get-next status %s", ad.Status)
	}
	ad.UpdatedAt = time.Now()

	newKey, err := makeAuctionDealKey(ad.ID, ad.Status)
	if err != nil {
		return AuctionDeal{}, false, fmt.Errorf("creating new key: %s", err)
	}
	if err := save(txn, newKey, ad); err != nil {
		return AuctionDeal{}, false, fmt.Errorf("save moved auction deal: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return AuctionDeal{}, false, fmt.Errorf("committing txn: %s", err)
	}

	return ad, true, nil
}

// SaveAndMoveAuctionDeal persists a modified auction deal, and move it to the provided status.
func (s *Store) SaveAndMoveAuctionDeal(aud AuctionDeal, newStatus AuctionDealStatus) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %s", err)
	}
	defer txn.Discard()

	if err := areAllExpectedFieldsSet(aud); err != nil {
		return fmt.Errorf("validating that expected fields are set for new status: %s", err)
	}

	currentKey, err := makeAuctionDealKey(aud.ID, aud.Status)
	if err != nil {
		return fmt.Errorf("generating current datastore key %s %s", aud.ID, aud.Status)
	}

	exists, err := txn.Has(currentKey)
	if err != nil {
		return fmt.Errorf("checking key %s: %s", currentKey, err)
	}
	if !exists {
		return ErrNotFound
	}

	// Remove from current datastore namespace.
	if err := txn.Delete(currentKey); err != nil {
		return fmt.Errorf("deleting current record: %s", err)
	}

	aud.Status = newStatus
	aud.UpdatedAt = time.Now()

	// Save again in the new datastore namespace.
	newKey, err := makeAuctionDealKey(aud.ID, aud.Status)
	if err != nil {
		return fmt.Errorf("generating new auction deal key %s %s", aud.ID, aud.Status)
	}
	if err := save(txn, newKey, aud); err != nil {
		return fmt.Errorf("saving auction deal status change: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// GetAuctionData returns an auction data by id. If not found returns ErrNotFound.
func (s *Store) GetAuctionData(auctionDataID string) (AuctionData, error) {
	var ad AuctionData
	if err := get(s.ds, makeAuctionDataKey(auctionDataID), &ad); err != nil {
		return AuctionData{}, err
	}

	return ad, nil
}

// RemoveAuctionDeal removes the provided auction deal. If the corresponding auction data isn't linked
// with any remaining auction deals, then is also removed.
func (s *Store) RemoveAuctionDeal(aud AuctionDeal) error {
	if aud.Status != ExecutingReportFinalized {
		return fmt.Errorf("only auction deals in final status can be removed")
	}

	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %s", err)
	}
	defer txn.Discard()

	// 1. Remove the auction deal.
	key, err := makeAuctionDealKey(aud.ID, aud.Status)
	if err != nil {
		return fmt.Errorf("generating auction deal key %s %s", aud.ID, aud.Status)
	}
	if err := s.ds.Delete(key); err != nil {
		return fmt.Errorf("deleting auction deal: %s", err)
	}

	// 2. Get the related AuctionData. If after decreased the counter of linked AuctionDeals
	//    we get 0, then proceed to also delete it (since nobody will use it again).
	var ad AuctionData
	adKey := makeAuctionDataKey(aud.AuctionDataID)
	if err := get(txn, adKey, &ad); err != nil {
		return fmt.Errorf("get linked auction data: %s", err)
	}
	ad.LinkedAuctionDeals--
	if ad.LinkedAuctionDeals == 0 {
		if err := txn.Delete(adKey); err != nil {
			return fmt.Errorf("deleting orphaned auction data: %s", err)
		}
	} else {
		if err := save(txn, adKey, ad); err != nil {
			return fmt.Errorf("saving updated auction data: %s", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing txn: %s", err)
	}

	return nil
}

func get(dsRead datastore.Read, key datastore.Key, b interface{}) error {
	buf, err := dsRead.Get(key)
	if err == datastore.ErrNotFound {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get value from datastore: %s", err)
	}
	d := gob.NewDecoder(bytes.NewReader(buf))
	if err := d.Decode(b); err != nil {
		return fmt.Errorf("unmarshaling gob: %s", err)
	}
	return nil
}

func save(dsWrite datastore.Write, key datastore.Key, b interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(b); err != nil {
		return fmt.Errorf("gob encoding: %s", err)
	}
	if err := dsWrite.Put(key, buf.Bytes()); err != nil {
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

func validate(ad *AuctionData, ads []*AuctionDeal) error {
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

func areAllExpectedFieldsSet(ad AuctionDeal) error {
	switch ad.Status {
	case PendingDealMaking, ExecutingDealMaking:
		// Nothing to validate.
	case PendingConfirmation, ExecutingConfirmation:
		if !ad.ProposalCid.Defined() {
			return errors.New("proposal cid should be set to transition to WaitingConfirmation")
		}
	case PendingReportFinalized:
		if ad.ErrorCause == "" { // Success
			if ad.DealID == 0 {
				return errors.New("a success status should have a defined deal-id")
			}
			if ad.DealExpiration == 0 {
				return errors.New("a success status should have a defined deal-expiration")
			}
			if ad.ErrorCause != "" {
				return fmt.Errorf("a success status can't have an error cause: %s", ad.ErrorCause)
			}
		}
	default:
		return fmt.Errorf("unknown status: %s", ad.Status)
	}
	return nil
}

func makeAuctionDataKey(id string) datastore.Key {
	return dsPrefixAuctionData.ChildString(id)
}

func makeAuctionDealKey(id string, status AuctionDealStatus) (datastore.Key, error) {
	prefix, err := getAuctionDealDSPrefix(status)
	if err != nil {
		return datastore.Key{}, err
	}
	return prefix.ChildString(id), nil
}

func getAuctionDealDSPrefix(status AuctionDealStatus) (datastore.Key, error) {
	switch status {
	case PendingDealMaking:
		return dsPrefixAuctionDealPendingDealMaking, nil
	case ExecutingDealMaking:
		return dsPrefixAuctionDealExecutingDealMaking, nil
	case PendingConfirmation:
		return dsPrefixAuctionDealPendingConfirmation, nil
	case ExecutingConfirmation:
		return dsPrefixAuctionDealExecutingConfirmation, nil
	case PendingReportFinalized:
		return dsPrefixAuctionPendingReportFinalized, nil
	case ExecutingReportFinalized:
		return dsPrefixAuctionExecutingReportFinalized, nil

	default:
		return datastore.Key{}, fmt.Errorf("unkown status: %s", status)
	}
}

func (ads AuctionDealStatus) String() string {
	switch ads {
	case PendingDealMaking:
		return "PendingDealMaking"
	case ExecutingDealMaking:
		return "ExecutingDealMaking"
	case PendingConfirmation:
		return "PendingConfirmation"
	case ExecutingConfirmation:
		return "ExecutingConfirmation"
	case PendingReportFinalized:
		return "PendingReportFinalized"
	case ExecutingReportFinalized:
		return "ExecutingReportFinalied"
	default:
		panic("unknown deal status")
	}
}

// getAll is a method only to be used in tests.
func (s *Store) getAll(status AuctionDealStatus) ([]AuctionDeal, error) {
	dsPrefix, err := getAuctionDealDSPrefix(status)
	if err != nil {
		return nil, fmt.Errorf("get auction deal datastore prefix: %s", err)
	}

	q := query.Query{Prefix: dsPrefix.String()}
	res, err := s.ds.Query(q)
	if err != nil {
		return nil, fmt.Errorf("creating query: %s", err)
	}
	defer res.Close()

	var ads []AuctionDeal
	for item := range res.Next() {
		if item.Error != nil {
			return nil, fmt.Errorf("get query item: %s", item.Error)
		}
		d := gob.NewDecoder(bytes.NewReader(item.Value))
		var ad AuctionDeal
		if err := d.Decode(&ad); err != nil {
			return nil, fmt.Errorf("unmarshaling gob: %s", err)
		}
		ads = append(ads, ad)
	}

	return ads, nil
}
