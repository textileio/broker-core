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

func (s *Store) CreateOrUpdateAuction(ctx context.Context, a *pb.AuctionSummary) (err error) {
	return s.db.CreateOrUpdateAuction(ctx, db.CreateOrUpdateAuctionParams{
		a.Id,
		a.BatchId,
		a.DealVerified,
		a.ExcludedStorageProviders,
		db.AuctionStatus(a.Status),
		a.StartedAt.AsTime(),
		a.UpdatedAt.AsTime(),
		int64(a.Duration),
		"", //a.ErrorCause,
	})
}

func (s *Store) CreateBid(ctx context.Context, auctionID string, b *auctioneer.Bid) (err error) {
	return s.db.CreateBid(ctx, db.CreateBidParams{
		auctionID,
		b.MinerAddr,
		b.WalletAddrSig,
		b.BidderID.String(),
		b.AskPrice,
		b.VerifiedAskPrice,
		int64(b.StartEpoch),
		b.FastRetrieval,
		b.ReceivedAt,
	})
}

func (s *Store) WonBid(ctx context.Context, auctionID string, b *auctioneer.Bid, t time.Time) (err error) {
	return s.db.WonBid(ctx, db.WonBidParams{
		AuctionID: auctionID,
		BidderID:  b.BidderID.String(),
		WonAt:     sql.NullTime{Time: t, Valid: true},
	})
}

func (s *Store) AckedBid(ctx context.Context, auctionID string, b *auctioneer.Bid, t time.Time) (err error) {
	return s.db.AcknowledgedBid(ctx, db.AcknowledgedBidParams{
		AuctionID:      auctionID,
		BidderID:       b.BidderID.String(),
		AcknowledgedAt: sql.NullTime{Time: t, Valid: true},
	})
}

func (s *Store) ProposalDelivered(ctx context.Context, ts time.Time, auctionID,
	bidID, proposalCid, errorCause string) (err error) {
	return s.db.ProposalDelivered(ctx, db.ProposalDeliveredParams{
		AuctionID: auctionID,
		// 	BidderID:  b.BidderID.String(),
		ProposalCid:            sql.NullString{String: proposalCid, Valid: true},
		ProposalCidDeliveredAt: sql.NullTime{Time: ts, Valid: true},
	})
}

func (s *Store) AuctionClosed(ctx context.Context, ca broker.ClosedAuction, ts time.Time) error {
	return s.db.CloseAuction(ctx, db.CloseAuctionParams{
		ID:       string(ca.ID),
		Status:   db.AuctionStatus(ca.Status.String()),
		ClosedAt: sql.NullTime{Time: ts, Valid: true},
	})
}
