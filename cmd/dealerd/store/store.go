package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/ipfs/go-cid"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/dealerd/store/internal/db"
	"github.com/textileio/broker-core/cmd/dealerd/store/migrations"
	"github.com/textileio/broker-core/storeutil"
)

const (
	// StatusDealMaking indicates that an auction data is ready for deal making.
	StatusDealMaking = AuctionDealStatus(db.StatusDealMaking)
	// StatusConfirmation indicates that a deal is pending to be confirmed.
	StatusConfirmation = AuctionDealStatus(db.StatusConfirmation)
	// StatusReportFinalized indicates that a deal result is pending to be reported.
	StatusReportFinalized = AuctionDealStatus(db.StatusReportFinalized)
)

var (
	// ErrNotFound is returned if the item isn't found in the store.
	ErrNotFound = errors.New("key not found")
)

// AuctionDealStatus is the type of action deal status.
type AuctionDealStatus db.Status

// AuctionData contains information of data to be stored in Filecoin.
type AuctionData struct {
	ID            string
	StorageDealID broker.StorageDealID
	PayloadCid    cid.Cid
	PieceCid      cid.Cid
	PieceSize     uint64
	Duration      uint64
	CreatedAt     time.Time
}

// AuctionDeal contains information to make a deal with a particular miner. The data information is stored
// in the linked AuctionData.
type AuctionDeal db.AuctionDeal

// Store provides persistent storage for Bids.
type Store struct {
	conn    *sql.DB
	db      *db.Queries
	lock    sync.Mutex
	entropy *ulid.MonotonicEntropy
}

// New returns a *Store.
func New(postgresURI string) (*Store, error) {
	as := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})
	conn, err := storeutil.MigrateAndConnectToDB(postgresURI, as)
	if err != nil {
		return nil, err
	}
	return &Store{conn: conn, db: db.New(conn)}, nil
}

// CtxWithTx attach a database transaction to the context. It returns the
// context unchanged if there's error starting the transaction.
func (s *Store) CtxWithTx(ctx context.Context, opts ...storeutil.TxOptions) (context.Context, error) {
	return storeutil.CtxWithTx(ctx, s.conn, opts...)
}

// FinishTxForCtx commits or rolls back the transaction attatched to the
// context depending on the error passed in and returns the result. It errors
// if the context doesn't have a transaction attached.
func (s *Store) FinishTxForCtx(ctx context.Context, err error) error {
	return storeutil.FinishTxForCtx(ctx, err)
}

//nolint:unparam
func (s *Store) withTx(ctx context.Context, f func(*db.Queries) error, opts ...storeutil.TxOptions) (err error) {
	return storeutil.WithTx(ctx, s.conn, func(tx *sql.Tx) error {
		return f(s.db.WithTx(tx))
	}, opts...)
}

func (s *Store) useTxFromCtx(ctx context.Context, f func(*db.Queries) error) (err error) {
	return storeutil.UseTxFromCtx(ctx,
		func(tx *sql.Tx) error { return f(s.db.WithTx(tx)) },
		func() error { return f(s.db) })
}

// Create persist new auction data and a set of related auction deals. It
// writes the IDs back to the parameters passed in.
func (s *Store) Create(ctx context.Context, ad *AuctionData, ads []*AuctionDeal) error {
	err := validate(ad, ads)
	if err != nil {
		return fmt.Errorf("invalid auction data: %s", err)
	}

	if ad.ID, err = s.newID(); err != nil {
		return fmt.Errorf("generating new id: %s", err)
	}
	return s.withTx(ctx, func(txn *db.Queries) error {
		if err := txn.CreateAuctionData(ctx, db.CreateAuctionDataParams{
			ID:            ad.ID,
			StorageDealID: ad.StorageDealID,
			PayloadCid:    ad.PayloadCid.String(),
			PieceCid:      ad.PieceCid.String(),
			PieceSize:     ad.PieceSize,
			Duration:      ad.Duration,
		}); err != nil {
			return fmt.Errorf("saving auction data in datastore: %s", err)
		}
		for _, deal := range ads {
			deal.ID, err = s.newID()
			if err != nil {
				return fmt.Errorf("generating new id: %s", err)
			}
			deal.AuctionDataID = ad.ID
			deal.Status = db.StatusDealMaking
			deal.ReadyAt = time.Unix(0, 0)
			if err := txn.CreateAuctionDeal(ctx, db.CreateAuctionDealParams{
				ID:                  deal.ID,
				AuctionDataID:       deal.AuctionDataID,
				MinerID:             deal.MinerID,
				PricePerGibPerEpoch: deal.PricePerGibPerEpoch,
				StartEpoch:          deal.StartEpoch,
				Verified:            deal.Verified,
				FastRetrieval:       deal.FastRetrieval,
				AuctionID:           deal.AuctionID,
				BidID:               deal.BidID,
				Status:              deal.Status,
				Executing:           deal.Executing,
				ErrorCause:          deal.ErrorCause,
				Retries:             deal.Retries,
				ProposalCid:         deal.ProposalCid,
				DealID:              deal.DealID,
				DealExpiration:      deal.DealExpiration,
				DealMarketStatus:    deal.DealMarketStatus,
				ReadyAt:             deal.ReadyAt,
			}); err != nil {
				return fmt.Errorf("saving auction deal in datastore: %s", err)
			}
		}
		return nil
	})
}

// GetNextPending returns the next auction deal in the provided status and marks
// the deal as executing. If none exists, it returns false in the second parameter.
func (s *Store) GetNextPending(ctx context.Context, status AuctionDealStatus) (ad AuctionDeal, exists bool, err error) {
	err = s.useTxFromCtx(ctx, func(q *db.Queries) error {
		deal, err := q.NextPendingAuctionDeal(ctx, db.Status(status))
		if err == nil {
			ad = AuctionDeal(deal)
		}
		return err
	})
	if err == sql.ErrNoRows {
		return AuctionDeal{}, false, nil
	} else if err != nil {
		return
	}
	return ad, true, nil
}

// SaveAndMoveAuctionDeal persists a modified auction deal, moves it to the
// provided status and marks it as not executing.
func (s *Store) SaveAndMoveAuctionDeal(ctx context.Context, aud AuctionDeal, newStatus AuctionDealStatus) error {
	if err := areAllExpectedFieldsSet(aud); err != nil {
		return fmt.Errorf("validating that expected fields are set for new status: %s", err)
	}
	return s.useTxFromCtx(ctx, func(q *db.Queries) error {
		rows, err := q.UpdateAuctionDeal(ctx, db.UpdateAuctionDealParams{
			ID:                  aud.ID,
			AuctionDataID:       aud.AuctionDataID,
			MinerID:             aud.MinerID,
			PricePerGibPerEpoch: aud.PricePerGibPerEpoch,
			StartEpoch:          aud.StartEpoch,
			Verified:            aud.Verified,
			FastRetrieval:       aud.FastRetrieval,
			AuctionID:           aud.AuctionID,
			BidID:               aud.BidID,

			Status:     db.Status(newStatus),
			Executing:  false,
			ErrorCause: aud.ErrorCause,
			Retries:    aud.Retries,

			ProposalCid:      aud.ProposalCid,
			DealID:           aud.DealID,
			DealExpiration:   aud.DealExpiration,
			DealMarketStatus: aud.DealMarketStatus,
			ReadyAt:          aud.ReadyAt,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// GetAuctionData returns an auction data by id.
func (s *Store) GetAuctionData(ctx context.Context, auctionDataID string) (ad AuctionData, err error) {
	err = s.useTxFromCtx(ctx, func(q *db.Queries) error {
		datum, err := q.GetAuctionData(ctx, auctionDataID)
		if err == nil {
			payloadCid, err := cid.Parse(datum.PayloadCid)
			if err != nil {
				return fmt.Errorf("parsing payload cid from db: %s", err)
			}
			pieceCid, err := cid.Parse(datum.PieceCid)
			if err != nil {
				return fmt.Errorf("parsing piece cid from db: %s", err)
			}
			ad = AuctionData{
				ID:            datum.ID,
				StorageDealID: datum.StorageDealID,
				PayloadCid:    payloadCid,
				PieceCid:      pieceCid,
				PieceSize:     datum.PieceSize,
				Duration:      datum.Duration,
				CreatedAt:     datum.CreatedAt,
			}
		}
		return err
	})
	return
}

// RemoveAuctionDeal removes the provided auction deal. If the corresponding auction data isn't linked
// with any remaining auction deals, then is also removed.
func (s *Store) RemoveAuctionDeal(ctx context.Context, aud AuctionDeal) error {
	if aud.Status != db.StatusReportFinalized || !aud.Executing {
		return fmt.Errorf("only auction deals in final status can be removed")
	}
	return s.withTx(ctx, func(txn *db.Queries) error {
		// 1. Remove the auction deal.
		if err := txn.RemoveAuctionDeal(ctx, aud.ID); err != nil {
			return fmt.Errorf("deleting auction deal: %s", err)
		}

		// 2. Get the related AuctionData. If after decreased the counter of linked AuctionDeals
		//    we get 0, then proceed to also delete it (since nobody will use it again).
		ids, err := txn.GetAuctionDealIDs(ctx, aud.AuctionDataID)
		if err != nil {
			return fmt.Errorf("get linked auction data: %s", err)
		}
		if len(ids) == 0 {
			if err := txn.RemoveAuctionData(ctx, aud.AuctionDataID); err != nil {
				return fmt.Errorf("deleting orphaned auction data: %s", err)
			}
		}
		return nil
	})
}

// GetStatusCounts returns a summary of in which deal statuses are watched deals.
func (s *Store) GetStatusCounts(ctx context.Context) (map[storagemarket.StorageDealStatus]int64, error) {
	ads, err := s.db.GetAuctionDealsByStatus(ctx, db.StatusConfirmation)
	if err != nil {
		return nil, fmt.Errorf("getting auction deals: %s", err)
	}
	ret := map[storagemarket.StorageDealStatus]int64{}
	for _, ad := range ads {
		ret[ad.DealMarketStatus]++
	}

	return ret, nil
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
		return errors.New("invalid payload cid")
	}
	if !ad.PieceCid.Defined() {
		return errors.New("invalid piece cid")
	}
	if ad.PieceSize <= 0 {
		return errors.New("piece size is zero")
	}

	for _, auctionDeal := range ads {
		if auctionDeal.MinerID == "" {
			return errors.New("miner address is empty")
		}
		if auctionDeal.PricePerGibPerEpoch < 0 {
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
	case db.StatusDealMaking:
		// Nothing to validate.
	case db.StatusConfirmation:
		proposalCid, err := cid.Parse(ad.ProposalCid)
		if err != nil {
			return fmt.Errorf("parse proposal cid: %s", err)
		}
		if !proposalCid.Defined() {
			return errors.New("proposal cid should be set to transition to WaitingConfirmation")
		}
	case db.StatusReportFinalized:
		if ad.Executing {
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
			return nil
		}
		fallthrough
	default:
		return fmt.Errorf("unknown status: %s", ad.Status) // and executing?
	}
	return nil
}

// getAllPending is a method only to be used in tests.
func (s *Store) getAllPending() (ret []AuctionDeal, err error) {
	ads, err := s.db.GetAuctionDealsByStatus(context.Background(), db.StatusDealMaking)
	if err != nil {
		return nil, fmt.Errorf("get auction deals: %s", err)
	}
	for _, ad := range ads {
		if !ad.Executing {
			ret = append(ret, AuctionDeal(ad))
		}
	}
	return
}
