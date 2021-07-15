package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" /*nolint*/
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/ipfs/go-cid"
	_ "github.com/jackc/pgx/v4/stdlib" /*nolint*/
	"github.com/multiformats/go-multiaddr"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/store/internal/db"
	"github.com/textileio/broker-core/cmd/brokerd/store/migrations"
	logger "github.com/textileio/go-log/v2"
)

var (
	// ErrNotFound is returned if the broker request doesn't exist.
	ErrNotFound = fmt.Errorf("not found")
	// ErrStorageDealContainsUnknownBrokerRequest is returned if a storage deal contains an
	// unknown broker request.
	ErrStorageDealContainsUnknownBrokerRequest = fmt.Errorf("storage deal contains an unknown broker request")

	log = logger.Logger("store")
)

// Store provides a persistent layer for broker requests.
type Store struct {
	conn    *sql.DB
	db      *db.Queries
	lock    sync.Mutex
	entropy *ulid.MonotonicEntropy
}

// New returns a new Store backed by `postgresURI`.
func New(postgresURI string) (*Store, error) {
	// To avoid dealing with time zone issues, we just enforce UTC timezone
	if !strings.Contains(postgresURI, "timezone=UTC") {
		return nil, errors.New("timezone=UTC is required in postgres URI")
	}
	s := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})
	d, err := bindata.WithInstance(s)
	if err != nil {
		return nil, err
	}
	m, err := migrate.NewWithSourceInstance("go-bindata", d, postgresURI)
	if err != nil {
		return nil, err
	}
	if err := m.Up(); err != nil {
		return nil, err
	}
	conn, err := sql.Open("pgx", postgresURI)
	if err != nil {
		return nil, err
	}
	return &Store{conn: conn, db: db.New(conn)}, nil
}

// CreateBrokerRequest creates the provided BrokerRequest in store.
func (s *Store) CreateBrokerRequest(ctx context.Context, br broker.BrokerRequest) error {
	return s.db.CreateBrokerRequest(ctx, db.CreateBrokerRequestParams{
		ID: br.ID, DataCid: br.DataCid.String(), Status: br.Status})
}

// GetBrokerRequest gets a BrokerRequest with the specified `id`. If not found returns ErrNotFound.
func (s *Store) GetBrokerRequest(ctx context.Context, id broker.BrokerRequestID) (broker.BrokerRequest, error) {
	br, err := s.db.GetBrokerRequest(ctx, id)
	if err == sql.ErrNoRows {
		return broker.BrokerRequest{}, ErrNotFound
	} else if err != nil {
		return broker.BrokerRequest{}, err
	}
	dataCid, err := cid.Parse(br.DataCid)
	if err != nil {
		return broker.BrokerRequest{}, err
	}
	return broker.BrokerRequest{
		ID:            br.ID,
		DataCid:       dataCid,
		Status:        br.Status,
		StorageDealID: broker.StorageDealID(br.StorageDealID.String),
		CreatedAt:     br.CreatedAt,
		UpdatedAt:     br.UpdatedAt,
	}, nil
}

// CreateStorageDeal persists a storage deal. It populates the sd.ID field with the corresponding id.
func (s *Store) CreateStorageDeal(ctx context.Context, sd *broker.StorageDeal, brIDs []broker.BrokerRequestID) error {
	if sd.ID == "" {
		return fmt.Errorf("storage deal id is empty")
	}
	var brStatus broker.BrokerRequestStatus
	// BrokerRequests status should mirror the StorageDeal status.
	switch sd.Status {
	case broker.StorageDealPreparing:
		brStatus = broker.RequestPreparing
	case broker.StorageDealAuctioning:
		brStatus = broker.RequestAuctioning
	default:
		return fmt.Errorf("unexpected storage deal initial status %d", sd.Status)
	}

	start := time.Now()
	defer log.Debugf(
		"creating storage deal %s with group size %d took %dms",
		sd.ID, len(brIDs),
		time.Since(start).Milliseconds(),
	)

	txn, err := s.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer txn.Rollback() // nolint:errcheck

	// 1- Get all involved BrokerRequests and validate that they exist.
	for _, brID := range brIDs {
		_, err := s.db.WithTx(txn).GetBrokerRequest(ctx, brID)
		if err == sql.ErrNoRows {
			return fmt.Errorf("unknown broker request id %s: %w", brID, ErrStorageDealContainsUnknownBrokerRequest)
		}
	}

	// 2- Persist the StorageDeal.
	dsources := struct {
		carURL         string
		ipfsCid        string
		ipfsMultiaddrs []string
	}{}
	if sd.Sources.CARURL != nil {
		dsources.carURL = sd.Sources.CARURL.URL.String()
	}
	if sd.Sources.CARIPFS != nil {
		dsources.ipfsCid = sd.Sources.CARIPFS.Cid.String()
		dsources.ipfsMultiaddrs = make([]string, len(sd.Sources.CARIPFS.Multiaddrs))
		for i, maddr := range sd.Sources.CARIPFS.Multiaddrs {
			dsources.ipfsMultiaddrs[i] = maddr.String()
		}
	}
	var pieceCid string
	if sd.PieceCid.Defined() {
		pieceCid = sd.PieceCid.String()
	}
	isd := db.CreateStorageDealParams{
		ID:                 sd.ID,
		Status:             sd.Status,
		RepFactor:          sd.RepFactor,
		DealDuration:       sd.DealDuration,
		PayloadCid:         sd.PayloadCid.String(),
		PieceCid:           pieceCid,
		PieceSize:          sd.PieceSize,
		DisallowRebatching: sd.DisallowRebatching,
		AuctionRetries:     sd.AuctionRetries,
		FilEpochDeadline:   sd.FilEpochDeadline,
		CarUrl:             dsources.carURL,
		CarIpfsCid:         dsources.ipfsCid,
		CarIpfsAddrs:       strings.Join(dsources.ipfsMultiaddrs, ","),
	}
	if err := s.db.WithTx(txn).CreateStorageDeal(ctx, isd); err != nil {
		return fmt.Errorf("saving storage deal: %s", err)
	}

	// 3- update the BrokerRequests with correct StorageDealID and status.
	ids := make([]string, len(brIDs))
	for i, id := range brIDs {
		ids[i] = string(id)
	}
	if err := s.db.WithTx(txn).UpdateBrokerRequests(ctx, db.UpdateBrokerRequestsParams{
		Column1: ids, Status: brStatus, StorageDealID: storageDealIDToSQL(sd.ID)}); err != nil {
		return fmt.Errorf("saving broker request: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// StorageDealToAuctioning moves a storage deal and the underlying broker requests
// to Auctioning status.
func (s *Store) StorageDealToAuctioning(
	ctx context.Context,
	id broker.StorageDealID,
	pieceCid cid.Cid,
	pieceSize uint64) error {
	txn, err := s.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer txn.Rollback() // nolint:errcheck

	sd, err := s.db.WithTx(txn).GetStorageDeal(ctx, id)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}

	// Take care of correct state transitions.
	switch sd.Status {
	case broker.StorageDealPreparing:
		if sd.PieceCid != "" || sd.PieceSize > 0 {
			return fmt.Errorf("piece cid and size should be empty: %s %d", sd.PieceCid, sd.PieceSize)
		}
	case broker.StorageDealAuctioning:
		if sd.PieceCid != pieceCid.String() {
			return fmt.Errorf("piececid different from registered: %s %s", sd.PieceCid, pieceCid)
		}
		if sd.PieceSize != pieceSize {
			return fmt.Errorf("piece size different from registered: %d %d", sd.PieceSize, pieceSize)
		}
		return nil
	default:
		return fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
	}

	if err := s.db.WithTx(txn).UpdateStorageDeal(ctx, db.UpdateStorageDealParams{
		ID:        sd.ID,
		Status:    broker.StorageDealAuctioning,
		PieceCid:  pieceCid.String(),
		PieceSize: pieceSize,
	}); err != nil {
		return fmt.Errorf("update storage deal: %s", err)
	}

	if err := s.db.WithTx(txn).UpdateBrokerRequestsStatus(ctx, db.UpdateBrokerRequestsStatusParams{
		Status: broker.RequestAuctioning, StorageDealID: storageDealIDToSQL(sd.ID)}); err != nil {
		return fmt.Errorf("update broker requests status: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// StorageDealError moves a storage deal to an error status with a specified error cause.
// The underlying broker requests are moved to Batching status. The caller is responsible to
// schedule again this broker requests to Packer.
func (s *Store) StorageDealError(
	ctx context.Context,
	id broker.StorageDealID,
	errorCause string,
	rebatch bool) ([]broker.BrokerRequestID, error) {
	txn, err := s.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer txn.Rollback() // nolint:errcheck

	sd, err := s.db.WithTx(txn).GetStorageDeal(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get storage deal: %s", err)
	}

	// 1. Verify some pre-state conditions.
	switch sd.Status {
	case broker.StorageDealAuctioning, broker.StorageDealDealMaking:
		if sd.Error != "" {
			return nil, fmt.Errorf("error cause should be empty: %s", sd.Error)
		}
	case broker.StorageDealError:
		if sd.Error != errorCause {
			return nil, fmt.Errorf("the error cause is different from the registeredon : %s %s", sd.Error, errorCause)
		}
		return s.db.WithTx(txn).GetBrokerRequestIDs(ctx, storageDealIDToSQL(id))
	default:
		return nil, fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
	}

	// 2. Move the StorageDeal to StorageDealError with the error cause.
	now := time.Now()
	sd.Status = broker.StorageDealError
	sd.Error = errorCause
	sd.UpdatedAt = now

	if err := s.db.WithTx(txn).UpdateStorageDealStatusAndError(ctx, db.UpdateStorageDealStatusAndErrorParams{
		ID: id, Error: errorCause, Status: broker.StorageDealError}); err != nil {
		return nil, fmt.Errorf("save storage deal: %s", err)
	}

	// 3. Move every underlying BrokerRequest to batching again, since they will be re-batched.
	status := broker.RequestError
	if rebatch {
		status = broker.RequestBatching
	}

	if err := s.db.WithTx(txn).UpdateBrokerRequestsStatus(ctx, db.UpdateBrokerRequestsStatusParams{
		StorageDealID: storageDealIDToSQL(id), Status: status}); err != nil {
		return nil, fmt.Errorf("getting broker request: %s", err)
	}
	if rebatch {
		if err := s.db.WithTx(txn).RebatchBrokerRequests(ctx, db.RebatchBrokerRequestsParams{
			StorageDealID: storageDealIDToSQL(id), ErrorCause: errorCause}); err != nil {
			return nil, fmt.Errorf("getting broker request: %s", err)
		}
	}

	// 4. Mark the batchCid as unpinnable, since it won't be used anymore for auctions or deals.
	unpinID, err := s.newID()
	if err != nil {
		return nil, fmt.Errorf("generating id for unpin job: %s", err)
	}

	if err := s.db.WithTx(txn).CreateUnpinJob(ctx, db.CreateUnpinJobParams{
		ID: unpinID, Cid: sd.PayloadCid, Type: int16(UnpinTypeBatch)}); err != nil {
		return nil, fmt.Errorf("saving unpin job: %s", err)
	}

	if err := txn.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %s", err)
	}

	return s.db.GetBrokerRequestIDs(ctx, storageDealIDToSQL(id))
}

// StorageDealSuccess moves a storage deal and the underlying broker requests to
// Success status.
func (s *Store) StorageDealSuccess(ctx context.Context, id broker.StorageDealID) error {
	txn, err := s.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer txn.Rollback() // nolint:errcheck

	sd, err := s.db.WithTx(txn).GetStorageDeal(ctx, id)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}

	// Take care of correct state transitions.
	switch sd.Status {
	case broker.StorageDealDealMaking:
		if sd.Error != "" {
			return fmt.Errorf("error cause should be empty: %s", sd.Error)
		}
	case broker.StorageDealSuccess:
		return nil
	default:
		return fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
	}

	if err := s.db.WithTx(txn).UpdateStorageDealStatus(ctx, db.UpdateStorageDealStatusParams{
		ID: id, Status: broker.StorageDealSuccess}); err != nil {
		return fmt.Errorf("updating storage deal status: %s", err)
	}

	if err := s.db.WithTx(txn).UpdateBrokerRequestsStatus(ctx, db.UpdateBrokerRequestsStatusParams{
		StorageDealID: storageDealIDToSQL(id), Status: broker.RequestSuccess}); err != nil {
		return fmt.Errorf("updating  broker requests status: %s", err)
	}

	brs, err := s.db.WithTx(txn).GetBrokerRequests(ctx, storageDealIDToSQL(id))
	if err != nil {
		return fmt.Errorf("getting broker requests: %s", err)
	}
	for _, br := range brs {
		unpinID, err := s.newID()
		if err != nil {
			return fmt.Errorf("generating id for unpin job: %s", err)
		}
		if err := s.db.WithTx(txn).CreateUnpinJob(ctx, db.CreateUnpinJobParams{
			ID: unpinID, Cid: br.DataCid, Type: int16(UnpinTypeData)}); err != nil {
			return fmt.Errorf("saving unpin job: %s", err)
		}
	}

	unpinID, err := s.newID()
	if err != nil {
		return fmt.Errorf("generating id for unpin job: %s", err)
	}
	if err := s.db.WithTx(txn).CreateUnpinJob(ctx, db.CreateUnpinJobParams{
		ID: unpinID, Cid: sd.PayloadCid, Type: int16(UnpinTypeBatch)}); err != nil {
		return fmt.Errorf("saving unpin job: %s", err)
	}
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// CountAuctionRetry increases the number of auction retries counter for a storage deal.
func (s *Store) CountAuctionRetry(ctx context.Context, id broker.StorageDealID) error {
	return s.db.ReauctionStorageDeal(ctx, id)
}

// AddMinerDeals includes new deals from a finalized auction.
func (s *Store) AddMinerDeals(ctx context.Context, auction broker.ClosedAuction) error {
	txn, err := s.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer txn.Rollback() // nolint:errcheck

	sd, err := s.db.WithTx(txn).GetStorageDeal(ctx, auction.StorageDealID)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}

	// Take care of correct state transitions.
	if sd.Status != broker.StorageDealAuctioning && sd.Status != broker.StorageDealDealMaking {
		return fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
	}

	// Add winning bids to list of deals.
	for bidID, bid := range auction.WinningBids {
		err := s.db.WithTx(txn).CreateMinerDeal(ctx, db.CreateMinerDealParams{
			StorageDealID: auction.StorageDealID,
			AuctionID:     auction.ID,
			BidID:         bidID,
			MinerAddr:     bid.MinerAddr,
		})
		if err != nil {
			return fmt.Errorf("save storage deal: %s", err)
		}
	}

	moveBrokerRequestsToDealMaking := false
	if sd.Status == broker.StorageDealAuctioning {
		if err := s.db.WithTx(txn).UpdateStorageDealStatus(ctx, db.UpdateStorageDealStatusParams{
			ID: auction.StorageDealID, Status: broker.StorageDealDealMaking}); err != nil {
			return fmt.Errorf("save storage deal: %s", err)
		}
		moveBrokerRequestsToDealMaking = true
	}

	// If the storage-deal was already in StorageDealDealMaking, then we're
	// just adding more winning bids from new auctions we created to statisfy the
	// replication factor. That case means we already updated the underlying
	// broker-requests status.
	// This conditional is to only move the broker-request on the first successful auction
	// that might happen. On further auctions, we don't need to do this again.
	if moveBrokerRequestsToDealMaking {
		if err := s.db.WithTx(txn).UpdateBrokerRequestsStatus(ctx, db.UpdateBrokerRequestsStatusParams{
			StorageDealID: storageDealIDToSQL(sd.ID), Status: broker.RequestDealMaking}); err != nil {
			return fmt.Errorf("saving broker request: %s", err)
		}
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %s", err)
	}

	return nil
}

// GetStorageDeal gets an existing storage deal by id. If the storage deal doesn't exists, it returns
// ErrNotFound.
func (s *Store) GetStorageDeal(ctx context.Context, id broker.StorageDealID) (*broker.StorageDeal, error) {
	sd, err := s.db.GetStorageDeal(ctx, id)
	if err != nil {
		return nil, err
	}
	return storageDealFromDB(&sd)
}

// GetMinerDeals gets miner deals for a storage deal.
func (s *Store) GetMinerDeals(ctx context.Context, id broker.StorageDealID) ([]db.MinerDeal, error) {
	return s.db.GetMinerDeals(ctx, id)
}

// GetBrokerRequestIDs gets the ids of the broker requests for a storage deal.
func (s *Store) GetBrokerRequestIDs(ctx context.Context, id broker.StorageDealID) ([]broker.BrokerRequestID, error) {
	return s.db.GetBrokerRequestIDs(ctx, storageDealIDToSQL(id))
}

// SaveMinerDeals saves a new finalized (succeeded or errored) auction deal
// into the storage deal.
func (s *Store) SaveMinerDeals(ctx context.Context, fad broker.FinalizedAuctionDeal) error {
	rows, err := s.db.UpdateMinerDeals(ctx,
		db.UpdateMinerDealsParams{
			StorageDealID:  fad.StorageDealID,
			MinerAddr:      fad.Miner,
			DealExpiration: fad.DealExpiration,
			DealID:         fad.DealID,
			ErrorCause:     fad.ErrorCause,
		})
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("deal not found: %v", fad)
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

func storageDealFromDB(sd *db.StorageDeal) (sd2 *broker.StorageDeal, err error) {
	var payloadCid cid.Cid
	if sd.PayloadCid != "" {
		payloadCid, err = cid.Parse(sd.PayloadCid)
		if err != nil {
			return nil, fmt.Errorf("parsing payload CID: %s", err)
		}
	}
	var pieceCid cid.Cid
	if sd.PieceCid != "" {
		pieceCid, err = cid.Parse(sd.PieceCid)
		if err != nil {
			return nil, fmt.Errorf("parsing piece CID: %s", err)
		}
	}
	var sources auction.Sources
	if u, err := url.ParseRequestURI(sd.CarUrl); err == nil {
		sources.CARURL = &auction.CARURL{URL: *u}
	}
	if sd.CarIpfsCid != "" {
		if id, err := cid.Parse(sd.CarIpfsCid); err == nil {
			carIPFS := &auction.CARIPFS{Cid: id}
			addrs := strings.Split(sd.CarIpfsAddrs, ",")
			for _, addr := range addrs {
				if ma, err := multiaddr.NewMultiaddr(addr); err == nil {
					carIPFS.Multiaddrs = append(carIPFS.Multiaddrs, ma)
				} else {
					return nil, fmt.Errorf("parsing IPFS multiaddr: %s", err)
				}
			}
			sources.CARIPFS = carIPFS
		}
	}
	return &broker.StorageDeal{
		ID:                 sd.ID,
		Status:             sd.Status,
		RepFactor:          sd.RepFactor,
		DealDuration:       sd.DealDuration,
		PayloadCid:         payloadCid,
		PieceCid:           pieceCid,
		PieceSize:          sd.PieceSize,
		Sources:            sources,
		DisallowRebatching: sd.DisallowRebatching,
		AuctionRetries:     sd.AuctionRetries,
		FilEpochDeadline:   sd.FilEpochDeadline,
		Error:              sd.Error,
		CreatedAt:          sd.CreatedAt,
		UpdatedAt:          sd.UpdatedAt,
	}, nil
}

func storageDealIDToSQL(id broker.StorageDealID) sql.NullString {
	return sql.NullString{String: string(id), Valid: true}
}
