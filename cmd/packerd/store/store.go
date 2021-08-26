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

	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/ipfs/go-cid"
	"github.com/jackc/pgconn"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/packerd/store/internal/db"
	"github.com/textileio/broker-core/cmd/packerd/store/migrations"
	"github.com/textileio/broker-core/storeutil"
	logger "github.com/textileio/go-log/v2"
)

const (
	stuckSeconds = int64(3600)
)

var (
	log = logger.Logger("store")

	// ErrOperationIDExists indicates that the storage request inclusion
	// in a batch already exists.
	ErrOperationIDExists = errors.New("operation-id already exists")
)

// BatchStatus is the status of a batch.
type BatchStatus db.BatchStatus

const (
	// StatusOpen is an open batch.
	StatusOpen = BatchStatus(db.BatchStatusOpen)
	// StatusReady is a ready to be created batch.
	StatusReady = BatchStatus(db.BatchStatusReady)
	// StatusExecuting is a batch being processed.
	StatusExecuting = BatchStatus(db.BatchStatusExecuting)
	// StatusDone is an batch that was correctly created.
	StatusDone = BatchStatus(db.BatchStatusDone)
)

// StorageRequest is a storage request from a batch.
type StorageRequest = db.StorageRequest

// Store is a store for unprepared batches.
type Store struct {
	conn *sql.DB
	db   *db.Queries

	batchMaxSize int64
	batchMinSize int64

	lock    sync.Mutex
	entropy *ulid.MonotonicEntropy
}

// New returns a new Store.
func New(postgresURI string, batchMaxSize, batchMinSize int64) (*Store, error) {
	as := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})
	conn, err := storeutil.MigrateAndConnectToDB(postgresURI, as)
	if err != nil {
		return nil, fmt.Errorf("initializing db connection: %s", err)
	}

	s := &Store{
		conn:         conn,
		db:           db.New(conn),
		batchMaxSize: batchMaxSize,
		batchMinSize: batchMinSize,
	}

	return s, nil
}

// CtxWithTx attach a database transaction to the context. It returns the
// context unchanged if there's error starting the transaction.
func (s *Store) CtxWithTx(ctx context.Context, opts ...storeutil.TxOptions) (context.Context, error) {
	return storeutil.CtxWithTx(ctx, s.conn, opts...)
}

// AddStorageRequestToOpenBatch adds a storage request to an open batch if available, and
// creates one if that isn't the case.
func (s *Store) AddStorageRequestToOpenBatch(
	ctx context.Context,
	opID string,
	srID broker.StorageRequestID,
	dataCid cid.Cid,
	dataSize int64,
	origin string) error {
	if opID == "" {
		return errors.New("operation-id is empty")
	}
	if srID == "" {
		return errors.New("storage-request id is empty")
	}
	if !dataCid.Defined() {
		return errors.New("data-cid is undefined")
	}
	if dataSize == 0 {
		return errors.New("data-size is zero")
	}

	var ob db.Batch
	if err := storeutil.WithTx(ctx, s.conn, func(txn *sql.Tx) error {
		var err error
		queries := s.db.WithTx(txn)

		openBatchMaxSize := s.batchMaxSize - dataSize
		params := db.FindOpenBatchWithSpaceParams{TotalSize: openBatchMaxSize, Origin: origin}
		ob, err = queries.FindOpenBatchWithSpace(ctx, params)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("find open batch: %s", err)
		}

		if err == sql.ErrNoRows {
			log.Debugf("open batch from origin %s with max size %d not found, creating it", origin, openBatchMaxSize)
			newID, err := s.newID()
			if err != nil {
				return fmt.Errorf("create open batch id: %s", err)
			}
			newBatchID := broker.BatchID(newID)
			params := db.CreateOpenBatchParams{BatchID: newBatchID, Origin: origin}
			if err := queries.CreateOpenBatch(ctx, params); err != nil {
				return fmt.Errorf("creating open batch: %s", err)
			}
			ob = db.Batch{
				BatchID: newBatchID,
				Status:  db.BatchStatusOpen,
				Origin:  origin,
			}
		}

		asribParams := db.AddStorageRequestInBatchParams{
			OperationID:      opID,
			StorageRequestID: srID,
			DataCid:          dataCid.String(),
			BatchID:          ob.BatchID,
			Size:             dataSize,
		}
		if err := queries.AddStorageRequestInBatch(ctx, asribParams); err != nil {
			if err, ok := err.(*pgconn.PgError); ok {
				if err.Code == "23505" {
					return ErrOperationIDExists
				}
			}
			return fmt.Errorf("add storage request in batch: %w", err)
		}

		ubsParams := db.UpdateBatchSizeParams{
			BatchID:   ob.BatchID,
			TotalSize: ob.TotalSize + dataSize,
		}
		if err := queries.UpdateBatchSize(ctx, ubsParams); err != nil {
			return fmt.Errorf("update batch size: %s", err)
		}

		if ubsParams.TotalSize >= s.batchMinSize {
			mbtsParams := db.MoveBatchToStatusParams{
				BatchID: ob.BatchID,
				Status:  db.BatchStatusReady,
				ReadyAt: time.Now(),
			}
			if _, err := queries.MoveBatchToStatus(ctx, mbtsParams); err != nil {
				return fmt.Errorf("move batch to status: %s", err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("executing tx: %w", err)
	}

	log.Debugf("opID %s, storage-request %s, data-cid %s included in batch %s", opID, srID, dataCid, ob.BatchID)

	return nil
}

// MoveBatchToStatus moves a batch to a specified status.
func (s *Store) MoveBatchToStatus(
	ctx context.Context,
	batchID broker.BatchID,
	delay time.Duration,
	status BatchStatus) error {
	dbStatus, err := statusToDB(status)
	if err != nil {
		return fmt.Errorf("casting status to db type: %s", err)
	}
	params := db.MoveBatchToStatusParams{
		BatchID: batchID,
		Status:  dbStatus,
		ReadyAt: time.Now().Add(delay),
	}

	if err := s.withCtxTx(ctx, func(q *db.Queries) error {
		count, err := s.db.MoveBatchToStatus(ctx, params)
		if err != nil {
			return fmt.Errorf("move batch to status: %s", err)
		}
		if count != 1 {
			return fmt.Errorf("unexpected update count, got: %d, expected: 1", count)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// GetNextReadyBatch returns the next ready batch to be processed batch and changes the
// status to Executing.
// The caller is responsible for updating the status later to Ready on error, or Done on success.
func (s *Store) GetNextReadyBatch(
	ctx context.Context) (
	batchID broker.BatchID,
	totalSize int64,
	srs []StorageRequest,
	origin string,
	exists bool,
	err error) {
	if err = s.withCtxTx(ctx, func(q *db.Queries) error {
		var rb db.GetNextReadyBatchRow
		rb, err = s.db.GetNextReadyBatch(ctx, stuckSeconds)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return fmt.Errorf("db get next ready: %s", err)
		}
		if int64(time.Since(rb.ReadyAt).Seconds()) > stuckSeconds {
			log.Warnf("re-packing stuck batch %s", rb.BatchID)
		}

		srs, err = s.db.GetStorageRequestsFromBatch(ctx, rb.BatchID)
		if err != nil {
			return fmt.Errorf("get storage requests from batch: %s", err)
		}

		batchID = rb.BatchID
		totalSize = rb.TotalSize
		origin = rb.Origin
		exists = true
		return nil
	}); err != nil {
		return "", 0, nil, "", false, err
	}

	return
}

// DoneBatchStats provides information for a done batch.
type DoneBatchStats struct {
	Origin string
	Count  int64
	Bytes  int64
}

// OpenBatchStats provides information for an open batch.
type OpenBatchStats struct {
	Origin   string
	CidCount int64
	Bytes    int64
	Count    int64
}

// GetStats return stats about batches.
func (s *Store) GetStats(ctx context.Context) ([]OpenBatchStats, []DoneBatchStats, error) {
	openStats, err := s.db.OpenBatchStats(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("open batch stats stats: %s", err)
	}

	doneStats, err := s.db.DoneBatchStats(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("done batch stats: %s", err)
	}

	retOpenBatches := make([]OpenBatchStats, len(openStats))
	for i, stat := range openStats {
		retOpenBatches[i] = OpenBatchStats{
			Origin:   stat.Origin,
			CidCount: stat.CidCount,
			Count:    stat.BatchesCount,
			Bytes:    stat.Bytes,
		}
	}

	retDoneBatches := make([]DoneBatchStats, len(doneStats))
	for i, stat := range doneStats {
		retDoneBatches[i] = DoneBatchStats{
			Origin: stat.Origin,
			Count:  stat.BatchesCount,
			Bytes:  stat.Bytes,
		}
	}

	return retOpenBatches, retDoneBatches, nil
}

// Close closes the store.
func (s *Store) Close() error {
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("closing sql connection: %s", err)
	}
	return nil
}

func (s *Store) newID() (string, error) {
	s.lock.Lock()
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

func (s *Store) withCtxTx(ctx context.Context, f func(*db.Queries) error) error {
	return storeutil.WithCtxTx(ctx,
		func(tx *sql.Tx) error { return f(s.db.WithTx(tx)) },
		func() error { return f(s.db) })
}

func statusToDB(status BatchStatus) (db.BatchStatus, error) {
	switch status {
	case StatusOpen:
		return db.BatchStatusOpen, nil
	case StatusReady:
		return db.BatchStatusReady, nil
	case StatusExecuting:
		return db.BatchStatusExecuting, nil
	case StatusDone:
		return db.BatchStatusDone, nil
	default:
		return "", fmt.Errorf("unknown status: %#v", status)
	}
}
