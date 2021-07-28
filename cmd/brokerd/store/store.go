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

	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/ipfs/go-cid"
	"github.com/jackc/pgconn"
	"github.com/multiformats/go-multiaddr"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/store/internal/db"
	"github.com/textileio/broker-core/cmd/brokerd/store/migrations"
	"github.com/textileio/broker-core/storeutil"
	logger "github.com/textileio/go-log/v2"
)

var (
	// ErrNotFound is returned if the storage request doesn't exist.
	ErrNotFound = fmt.Errorf("not found")

	// ErrBatchExists if the provided batch id already exists.
	ErrBatchExists = errors.New("batch-id already exists")

	// ErrBatchInAuction if the provided batch id is already in auction status.
	ErrBatchInAuction = errors.New("batch-id already in auction")

	// ErrBatchContainsUnknownStorageRequest is returned if a batch contains an
	// unknown storage request.
	ErrBatchContainsUnknownStorageRequest = fmt.Errorf("batch contains an unknown storage request")

	log = logger.Logger("store")
)

// Store provides a persistent layer for storage requests.
type Store struct {
	conn    *sql.DB
	db      *db.Queries
	lock    sync.Mutex
	entropy *ulid.MonotonicEntropy
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

func (s *Store) withCtxTx(ctx context.Context, f func(*db.Queries) error) (err error) {
	return storeutil.WithCtxTx(ctx,
		func(tx *sql.Tx) error { return f(s.db.WithTx(tx)) },
		func() error { return f(s.db) })
}

// CreateStorageRequest creates the provided StorageRequest in store.
func (s *Store) CreateStorageRequest(ctx context.Context, br broker.StorageRequest) error {
	return s.withCtxTx(ctx, func(q *db.Queries) error {
		return q.CreateStorageRequest(ctx,
			db.CreateStorageRequestParams{
				ID:      br.ID,
				DataCid: br.DataCid.String(),
				Status:  br.Status,
				Origin:  br.Origin,
			})
	})
}

// GetStorageRequest gets a StorageRequest with the specified `id`. If not found returns ErrNotFound.
func (s *Store) GetStorageRequest(
	ctx context.Context,
	id broker.StorageRequestID) (br broker.StorageRequest, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		r, err := s.db.GetStorageRequest(ctx, id)
		if err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if err != nil {
			return err
		}

		dataCid, err := cid.Parse(r.DataCid)
		if err != nil {
			return err
		}
		br = broker.StorageRequest{
			ID:        r.ID,
			DataCid:   dataCid,
			Status:    r.Status,
			Origin:    r.Origin,
			BatchID:   broker.BatchID(r.BatchID.String),
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		}
		return err
	})
	return
}

// CreateBatch persists a batch.
func (s *Store) CreateBatch(ctx context.Context, ba *broker.Batch, brIDs []broker.StorageRequestID) error {
	if ba.ID == "" {
		return fmt.Errorf("batch id is empty")
	}
	var brStatus broker.StorageRequestStatus
	// StorageRequests status should mirror the Batch status.
	switch ba.Status {
	case broker.BatchStatusPreparing:
		brStatus = broker.RequestPreparing
	case broker.BatchStatusAuctioning:
		brStatus = broker.RequestAuctioning
	default:
		return fmt.Errorf("unexpected batch initial status %d", ba.Status)
	}

	start := time.Now()
	defer log.Debugf(
		"creating batch %s with group size %d took %dms",
		ba.ID, len(brIDs),
		time.Since(start).Milliseconds(),
	)

	// 2- Persist the Batch.
	dsources := struct {
		carURL         string
		ipfsCid        string
		ipfsMultiaddrs []string
	}{}
	if ba.Sources.CARURL != nil {
		dsources.carURL = ba.Sources.CARURL.URL.String()
	}
	if ba.Sources.CARIPFS != nil {
		dsources.ipfsCid = ba.Sources.CARIPFS.Cid.String()
		dsources.ipfsMultiaddrs = make([]string, len(ba.Sources.CARIPFS.Multiaddrs))
		for i, maddr := range ba.Sources.CARIPFS.Multiaddrs {
			dsources.ipfsMultiaddrs[i] = maddr.String()
		}
	}
	var pieceCid string
	if ba.PieceCid.Defined() {
		pieceCid = ba.PieceCid.String()
	}
	isd := db.CreateBatchParams{
		ID:                 ba.ID,
		Status:             ba.Status,
		RepFactor:          ba.RepFactor,
		DealDuration:       ba.DealDuration,
		PayloadCid:         ba.PayloadCid.String(),
		PieceCid:           pieceCid,
		PieceSize:          ba.PieceSize,
		DisallowRebatching: ba.DisallowRebatching,
		FilEpochDeadline:   ba.FilEpochDeadline,
		CarUrl:             dsources.carURL,
		CarIpfsCid:         dsources.ipfsCid,
		CarIpfsAddrs:       strings.Join(dsources.ipfsMultiaddrs, ","),
		Origin:             ba.Origin,
	}
	ids := make([]string, len(brIDs))
	for i, id := range brIDs {
		ids[i] = string(id)
	}

	return s.withTx(ctx, func(txn *db.Queries) error {
		if err := txn.CreateBatch(ctx, isd); err != nil {
			if err, ok := err.(*pgconn.PgError); ok {
				if err.Code == "23505" {
					return ErrBatchExists
				}
			}
			return fmt.Errorf("creating batch: %s", err)
		}
		updated, err := txn.BatchUpdateStorageRequests(ctx, db.BatchUpdateStorageRequestsParams{
			Ids: ids, Status: brStatus, BatchID: batchIDToSQL(ba.ID)})
		if err != nil {
			return fmt.Errorf("updating storage requests: %s", err)
		}
		// this check should be within the transaction to rollback when required
		if len(updated) < len(ids) {
			return fmt.Errorf("unknown storage request ids %v: %w",
				sliceDiff(ids, updated), ErrBatchContainsUnknownStorageRequest)
		}

		for key, value := range ba.Tags {
			if err := txn.CreateBatchTag(ctx, db.CreateBatchTagParams{
				BatchID: ba.ID,
				Key:     key,
				Value:   value,
			}); err != nil {
				return fmt.Errorf("creating tag: %s", err)
			}
		}

		return nil
	})
}

// BatchToAuctioning moves a batch and the underlying storage requests
// to Auctioning status.
func (s *Store) BatchToAuctioning(
	ctx context.Context,
	id broker.BatchID,
	pieceCid cid.Cid,
	pieceSize uint64) error {
	return s.withTx(ctx, func(txn *db.Queries) error {
		if err := txn.UpdateBatch(ctx, db.UpdateBatchParams{
			ID:        id,
			Status:    broker.BatchStatusAuctioning,
			PieceCid:  pieceCid.String(),
			PieceSize: pieceSize,
		}); err != nil {
			return fmt.Errorf("update batch: %s", err)
		}

		if err := txn.UpdateStorageRequestsStatus(ctx, db.UpdateStorageRequestsStatusParams{
			Status: broker.RequestAuctioning, BatchID: batchIDToSQL(id)}); err != nil {
			return fmt.Errorf("update storage requests status: %s", err)
		}
		return nil
	})
}

// BatchError moves a batch to an error status with a specified error cause.
// The underlying storage requests are moved to Batching status. The caller is responsible to
// schedule again this storage requests to Packer.
func (s *Store) BatchError(
	ctx context.Context,
	id broker.BatchID,
	errorCause string,
	rebatch bool) (brIDs []broker.StorageRequestID, err error) {
	err = s.withTx(ctx, func(txn *db.Queries) error {
		sd, err := txn.GetBatch(ctx, id)
		if err != nil {
			return fmt.Errorf("get batch: %s", err)
		}

		// 1. Verify some pre-state conditions.
		switch sd.Status {
		case broker.BatchStatusAuctioning, broker.BatchStatusDealMaking:
			if sd.Error != "" {
				return fmt.Errorf("error cause should be empty: %s", sd.Error)
			}
		case broker.BatchStatusError:
			if sd.Error != errorCause {
				return fmt.Errorf("the error cause is different from the registered on : %s %s", sd.Error, errorCause)
			}
			brIDs, err = txn.GetStorageRequestIDs(ctx, batchIDToSQL(id))
			return err
		default:
			return fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
		}

		// 2. Move the Batch to BatchError with the error cause.
		now := time.Now()
		sd.Status = broker.BatchStatusError
		sd.Error = errorCause
		sd.UpdatedAt = now

		if err := txn.UpdateBatchStatusAndError(ctx, db.UpdateBatchStatusAndErrorParams{
			ID: id, Error: errorCause, Status: broker.BatchStatusError}); err != nil {
			return fmt.Errorf("save batch: %s", err)
		}

		// 3. Move every underlying StorageRequest to batching again, since they will be re-batched.
		status := broker.RequestError
		if rebatch {
			status = broker.RequestBatching
		}

		if err := txn.UpdateStorageRequestsStatus(ctx, db.UpdateStorageRequestsStatusParams{
			BatchID: batchIDToSQL(id), Status: status}); err != nil {
			return fmt.Errorf("getting storage request: %s", err)
		}
		if rebatch {
			if err := txn.RebatchStorageRequests(ctx, db.RebatchStorageRequestsParams{
				BatchID: batchIDToSQL(id), ErrorCause: errorCause}); err != nil {
				return fmt.Errorf("getting storage request: %s", err)
			}
		}

		// 4. Mark the batchCid as unpinnable, since it won't be used anymore for auctions or deals.
		unpinID, err := s.newID()
		if err != nil {
			return fmt.Errorf("generating id for unpin job: %s", err)
		}

		if err := txn.CreateUnpinJob(ctx, db.CreateUnpinJobParams{
			ID: unpinID, Cid: sd.PayloadCid, Type: int16(UnpinTypeBatch)}); err != nil {
			return fmt.Errorf("saving unpin job: %s", err)
		}

		brIDs, err = txn.GetStorageRequestIDs(ctx, batchIDToSQL(id))
		return err
	})
	return
}

// BatchSuccess moves a batch and the underlying storage requests to
// Success status.
func (s *Store) BatchSuccess(ctx context.Context, id broker.BatchID) error {
	return s.withTx(ctx, func(txn *db.Queries) error {
		sd, err := txn.GetBatch(ctx, id)
		if err != nil {
			return fmt.Errorf("get batch: %s", err)
		}

		// Take care of correct state transitions.
		switch sd.Status {
		case broker.BatchStatusDealMaking:
			if sd.Error != "" {
				return fmt.Errorf("error cause should be empty: %s", sd.Error)
			}
		case broker.BatchStatusSuccess:
			return nil
		default:
			return fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
		}

		if err := txn.UpdateBatchStatus(ctx, db.UpdateBatchStatusParams{
			ID: id, Status: broker.BatchStatusSuccess}); err != nil {
			return fmt.Errorf("updating batch status: %s", err)
		}

		if err := txn.UpdateStorageRequestsStatus(ctx, db.UpdateStorageRequestsStatusParams{
			BatchID: batchIDToSQL(id), Status: broker.RequestSuccess}); err != nil {
			return fmt.Errorf("updating  storage requests status: %s", err)
		}

		brs, err := txn.GetStorageRequests(ctx, batchIDToSQL(id))
		if err != nil {
			return fmt.Errorf("getting storage requests: %s", err)
		}
		for _, br := range brs {
			unpinID, err := s.newID()
			if err != nil {
				return fmt.Errorf("generating id for unpin job: %s", err)
			}
			if err := txn.CreateUnpinJob(ctx, db.CreateUnpinJobParams{
				ID: unpinID, Cid: br.DataCid, Type: int16(UnpinTypeData)}); err != nil {
				return fmt.Errorf("saving unpin job: %s", err)
			}
		}

		unpinID, err := s.newID()
		if err != nil {
			return fmt.Errorf("generating id for unpin job: %s", err)
		}
		if err := txn.CreateUnpinJob(ctx, db.CreateUnpinJobParams{
			ID: unpinID, Cid: sd.PayloadCid, Type: int16(UnpinTypeBatch)}); err != nil {
			return fmt.Errorf("saving unpin job: %s", err)
		}
		return nil
	})
}

// AddDeals includes new deals from a finalized auction.
func (s *Store) AddDeals(ctx context.Context, auction broker.ClosedAuction) error {
	return s.withTx(ctx, func(txn *db.Queries) error {
		sd, err := txn.GetBatch(ctx, auction.BatchID)
		if err != nil {
			return fmt.Errorf("get batch: %s", err)
		}

		// Take care of correct state transitions.
		if sd.Status != broker.BatchStatusAuctioning && sd.Status != broker.BatchStatusDealMaking {
			return fmt.Errorf("wrong storage request status transition, tried moving to %s", sd.Status)
		}

		// Add winning bids to list of deals.
		for bidID, bid := range auction.WinningBids {
			if err := txn.CreateDeal(ctx, db.CreateDealParams{
				BatchID:           auction.BatchID,
				AuctionID:         auction.ID,
				BidID:             bidID,
				StorageProviderID: bid.StorageProviderID,
			}); err != nil {
				return fmt.Errorf("save batch: %s", err)
			}
		}

		moveStorageRequestsToDealMaking := false
		if sd.Status == broker.BatchStatusAuctioning {
			if err := txn.UpdateBatchStatus(ctx, db.UpdateBatchStatusParams{
				ID: auction.BatchID, Status: broker.BatchStatusDealMaking}); err != nil {
				return fmt.Errorf("save batch: %s", err)
			}
			moveStorageRequestsToDealMaking = true
		}

		// If the batch was already in BatchDealMaking, then we're
		// just adding more winning bids from new auctions we created to statisfy the
		// replication factor. That case means we already updated the underlying
		// storage-requests status.
		// This conditional is to only move the storage-request on the first successful auction
		// that might happen. On further auctions, we don't need to do this again.
		if moveStorageRequestsToDealMaking {
			if err := txn.UpdateStorageRequestsStatus(ctx, db.UpdateStorageRequestsStatusParams{
				BatchID: batchIDToSQL(sd.ID), Status: broker.RequestDealMaking}); err != nil {
				return fmt.Errorf("saving storage request: %s", err)
			}
		}

		return nil
	})
}

// GetBatch gets an existing batch by id. If the batch doesn't exists, it returns
// ErrNotFound.
func (s *Store) GetBatch(ctx context.Context, id broker.BatchID) (sd *broker.Batch, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		var dbSD db.Batch
		dbSD, err = q.GetBatch(ctx, id)
		if err != nil {
			return err
		}
		tags, err := q.GetBatchTags(ctx, id)
		if err != nil {
			return err
		}
		sd, err = batchFromDB(&dbSD, tags)
		return err
	})
	return
}

// GetDeals gets storage-provider deals for a batch.
func (s *Store) GetDeals(ctx context.Context, id broker.BatchID) (deals []db.Deal, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		deals, err = q.GetDeals(ctx, id)
		return err
	})
	return
}

// GetStorageRequestIDs gets the ids of the storage requests for a batch.
func (s *Store) GetStorageRequestIDs(ctx context.Context, id broker.BatchID) (
	brIDs []broker.StorageRequestID, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		brIDs, err = q.GetStorageRequestIDs(ctx, batchIDToSQL(id))
		return err
	})
	return
}

// OperationExists checks if the operation ID already exists in db.
func (s *Store) OperationExists(ctx context.Context, opID string) (exists bool, err error) {
	err = s.withCtxTx(ctx, func(q *db.Queries) error {
		err = q.CreateOperation(ctx, opID)
		if err, ok := err.(*pgconn.PgError); ok {
			if err.Code == "23505" {
				exists, err = true, nil
				return nil
			}
		}
		exists = false
		return err
	})
	return
}

// SaveDeals saves a new finalized (succeeded or errored) auction deal
// into the batch.
func (s *Store) SaveDeals(ctx context.Context, fad broker.FinalizedDeal) error {
	return s.withCtxTx(ctx, func(q *db.Queries) error {
		rows, err := q.UpdateDeals(ctx,
			db.UpdateDealsParams{
				BatchID:           fad.BatchID,
				StorageProviderID: fad.StorageProviderID,
				DealExpiration:    fad.DealExpiration,
				DealID:            fad.DealID,
				ErrorCause:        fad.ErrorCause,
			})
		if err != nil {
			return fmt.Errorf("update deal: %s", err)
		}
		if rows == 0 {
			return fmt.Errorf("deal not found: %v", fad)
		}
		return nil
	})
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

func batchFromDB(sd *db.Batch, tags []db.BatchTag) (sd2 *broker.Batch, err error) {
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
	mtags := make(map[string]string, len(tags))
	for _, tag := range tags {
		mtags[tag.Key] = tag.Value
	}

	return &broker.Batch{
		ID:                 sd.ID,
		Status:             sd.Status,
		RepFactor:          sd.RepFactor,
		DealDuration:       sd.DealDuration,
		PayloadCid:         payloadCid,
		PieceCid:           pieceCid,
		PieceSize:          sd.PieceSize,
		Sources:            sources,
		DisallowRebatching: sd.DisallowRebatching,
		FilEpochDeadline:   sd.FilEpochDeadline,
		Origin:             sd.Origin,
		Tags:               mtags,
		Error:              sd.Error,
		CreatedAt:          sd.CreatedAt,
		UpdatedAt:          sd.UpdatedAt,
	}, nil
}

func batchIDToSQL(id broker.BatchID) sql.NullString {
	return sql.NullString{String: string(id), Valid: true}
}

func sliceDiff(full []string, subset []broker.StorageRequestID) (diff []string) {
	fullSet := make(map[string]struct{}, len(full))
	for _, s := range full {
		fullSet[s] = struct{}{}
	}
	for _, s := range subset {
		if _, found := fullSet[string(s)]; !found {
			diff = append(diff, string(s))
		}
	}
	return
}
