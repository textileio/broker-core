package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/apid/store/internal/db"
	"github.com/textileio/broker-core/cmd/apid/store/migrations"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"github.com/textileio/broker-core/storeutil"
)

// Store provides a persistent layer for storage requests.
type Store struct {
	conn *sql.DB
	db   *db.Queries
}

// New returns a new Store backed by `postgresURI`.
func New(postgresURI string) (*Store, error) {
	as := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})
	conn, err := storeutil.MigrateAndConnectToDB(postgresURI, as)
	if err != nil {
		return nil, fmt.Errorf("initializing db connection: %s", err)
	}

	return &Store{conn: conn, db: db.New(conn)}, nil
}

// CreateOrUpdateAuction .
func (s *Store) CreateOrUpdateAuction(ctx context.Context, a *pb.AuctionSummary) (err error) {
	return s.db.CreateOrUpdateAuction(ctx, db.CreateOrUpdateAuctionParams{
		ID:                       a.Id,
		BatchID:                  a.BatchId,
		DealVerified:             a.DealVerified,
		ExcludedStorageProviders: a.ExcludedStorageProviders,
		Status:                   db.AuctionStatus(a.Status),
		StartedAt:                a.StartedAt.AsTime(),
		UpdatedAt:                a.UpdatedAt.AsTime(),
		Duration:                 int64(a.Duration),
		ErrorCause:               "",
	})
}

// CreateOrUpdateBid .
func (s *Store) CreateOrUpdateBid(ctx context.Context, auctionID string, b *auctioneer.Bid) (err error) {
	return s.db.CreateOrUpdateBid(ctx, db.CreateOrUpdateBidParams{
		AuctionID:         auctionID,
		StorageProviderID: b.MinerAddr,
		BidderID:          b.BidderID.String(),
		AskPrice:          b.AskPrice,
		VerifiedAskPrice:  b.VerifiedAskPrice,
		StartEpoch:        int64(b.StartEpoch),
		FastRetrieval:     b.FastRetrieval,
		ReceivedAt:        b.ReceivedAt,
	})
}

// WonBid .
func (s *Store) WonBid(ctx context.Context, auctionID string, b *auctioneer.Bid, t time.Time) (err error) {
	return s.db.WonBid(ctx, db.WonBidParams{
		AuctionID: auctionID,
		BidderID:  b.BidderID.String(),
		WonAt:     sql.NullTime{Time: t, Valid: true},
	})
}

// AckedBid .
func (s *Store) AckedBid(ctx context.Context, auctionID string, b *auctioneer.Bid, t time.Time) (err error) {
	return s.db.AcknowledgedBid(ctx, db.AcknowledgedBidParams{
		AuctionID:      auctionID,
		BidderID:       b.BidderID.String(),
		AcknowledgedAt: sql.NullTime{Time: t, Valid: true},
	})
}

// ProposalDelivered .
func (s *Store) ProposalDelivered(ctx context.Context, ts time.Time, auctionID,
	bidderID, bidID, proposalCid, errorCause string) (err error) {
	return s.db.ProposalDelivered(ctx, db.ProposalDeliveredParams{
		AuctionID:              auctionID,
		BidderID:               bidderID,
		ProposalCid:            sql.NullString{String: proposalCid, Valid: true},
		ProposalCidDeliveredAt: sql.NullTime{Time: ts, Valid: true},
	})
}

// AuctionClosed .
func (s *Store) AuctionClosed(ctx context.Context, ca broker.ClosedAuction, ts time.Time) error {
	return s.db.CloseAuction(ctx, db.CloseAuctionParams{
		ID:         string(ca.ID),
		Status:     db.AuctionStatus(ca.Status.String()),
		ClosedAt:   sql.NullTime{Time: ts, Valid: true},
		ErrorCause: ca.ErrorCause,
	})
}
