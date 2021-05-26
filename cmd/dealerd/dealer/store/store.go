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
	logger "github.com/ipfs/go-log/v2"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/dshelper/txndswrap"
)

var (
	// ErrNotFound is returned if the item isn't found in the store.
	ErrNotFound = errors.New("key isn't found")

	dsPrefixAuctionData                    = datastore.NewKey("/auction-data")
	dsPrefixAuctionDealPending             = datastore.NewKey("/auction-deal/pending")
	dsPrefixAuctionDealExecuting           = datastore.NewKey("/auction-deal/executing")
	dsPrefixAuctionDealWaitingConfirmation = datastore.NewKey("/auction-deal/waiting-confirm")
	dsPrefixAuctionDealFinalized           = datastore.NewKey("/auction-deal/finalized")

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
	// Pending indicates that an auction data is ready for deal making.
	Pending AuctionDealStatus = iota
	// Executing indicates that an auction is being executed.
	Executing
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
		ads[i].Status = Pending
		ads[i].CreatedAt = ad.CreatedAt
		ads[i].ReadyAt = time.Unix(0, 0)

		key, _ := makeAuctionDealKey(ads[i].ID, Pending)
		if err := save(txn, key, ads[i]); err != nil {
			return fmt.Errorf("saving auction deal in datastore: %s", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// SaveAndMoveStatusTo persist the auction deal information, and explicitely move its Status
// two the provided parameter. The Status field of the AuctionDeal will be overriden by the provided
// status, so callers shouldn't modify it before calling this method.
func (s *Store) SaveAndMoveStatusTo(aud AuctionDeal, newStatus AuctionDealStatus) error {
	txn, err := s.ds.NewTransaction(false)
	if err != nil {
		return fmt.Errorf("creating txn: %s", err)
	}
	defer txn.Discard()

	if err := isValidStatusChange(aud.Status, newStatus); err != nil {
		return fmt.Errorf("invalid status transition from %s to %s", aud.Status, newStatus)
	}
	if err := areAllExpectedFieldsSet(aud, newStatus); err != nil {
		return fmt.Errorf("validating that expected fields are set for new status: %s", err)
	}

	currentKey, err := makeAuctionDealKey(aud.ID, aud.Status)
	if err != nil {
		return fmt.Errorf("generating current datastore key %s %s", aud.ID, aud.Status)
	}
	// Remove from current datastore namespace.
	if err := txn.Delete(currentKey); err != nil {
		return fmt.Errorf("deleting current record: %s", err)
	}

	// Update values and new status.
	aud.ReadyAt = time.Unix(0, 0)
	aud.UpdatedAt = time.Now()
	aud.Status = newStatus

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
		return AuctionData{}, fmt.Errorf("get auction data from datastore: %s", err)
	}

	return ad, nil
}

// RemoveAuctionDeal removes the provided auction deal. If the corresponding auction data isn't linked
// with any remaining auction deals, then is also removed.
func (s *Store) RemoveAuctionDeal(aud AuctionDeal) error {
	if aud.Status != Error && aud.Status != Success {
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

func isValidStatusChange(pre AuctionDealStatus, post AuctionDealStatus) error {
	switch pre {
	case Pending:
		if post != Executing {
			return fmt.Errorf("expecting Executing but found: %s", post)
		}
	case Executing:
		if post != WaitingConfirmation {
			return fmt.Errorf("expecting WaitingConfirmation but found: %s", post)
		}

	case WaitingConfirmation:
		if post != Error && post != Success {
			return fmt.Errorf("expecting final status but found: %s", post)
		}
	case Success, Error:
		return fmt.Errorf("error/success status are final, so %s isn't allowed", post)
	default:
		return fmt.Errorf("unknown status: %s", post)
	}

	return nil
}

func areAllExpectedFieldsSet(ad AuctionDeal, statusCandidate AuctionDealStatus) error {
	switch statusCandidate {
	case Pending, Executing:
		// Nothing to validate.
	case WaitingConfirmation:
		if !ad.ProposalCid.Defined() {
			return errors.New("proposal cid should be set to transition to WaitingConfirmation")
		}
	case Success:
		if ad.DealID == 0 {
			return errors.New("a success status should have a defined deal-id")
		}
		if ad.DealExpiration == 0 {
			return errors.New("a success status should have a defined deal-expiration")
		}
		if ad.ErrorCause != "" {
			return fmt.Errorf("a success status can't have an error cause: %s", ad.ErrorCause)
		}
	case Error:
		if ad.ErrorCause == "" {
			return errors.New("an error status can't have an empty error cause")
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
	switch status {
	case Pending:
		return dsPrefixAuctionDealPending.ChildString(id), nil
	case Executing:

		return dsPrefixAuctionDealExecuting.ChildString(id), nil
	case WaitingConfirmation:
		return dsPrefixAuctionDealWaitingConfirmation.ChildString(id), nil
	case Success, Error:

		return dsPrefixAuctionDealFinalized.ChildString(id), nil
	default:
		return datastore.Key{}, fmt.Errorf("unkown status: %s", status)
	}
}

func (ads AuctionDealStatus) String() string {
	switch ads {
	case Pending:
		return "Pending"
	case Executing:
		return "Executing"
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
