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

var (
	log = logger.Logger("store")

	// ErrOperationIDExists indicates that the storage request inclusion
	// in a batch already exists.
	ErrOperationIDExists = errors.New("operation-id already exists")
)

// BatchStatus is the status of a batch
type BatchStatus int

const (
	// StatusOpen is an open batch.
	StatusOpen BatchStatus = iota
	// StatusReady is a ready to be created batch.
	StatusReady
	// StatusExecuting is a batch being processed.
	StatusExecuting
	// StatusDone is an batch that was correctly created.
	StatusDone
)

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

// CreateUnpreparedBatch creates a new pending unprepared batch to be prepared.
func (s *Store) AddStorageRequestToOpenBatch(ctx context.Context, opID string, srID broker.BrokerRequestID, dataCid cid.Cid, dataSize int64) error {
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
	if err := storeutil.MustUseTxFromCtx(ctx, func(txn *sql.Tx) error {
		var err error
		queries := s.db.WithTx(txn)

		openBatchMaxSize := s.batchMaxSize - dataSize
		ob, err = queries.FindOpenBatchWithSpace(ctx, openBatchMaxSize)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("find open batch: %s", err)
		}

		if err == sql.ErrNoRows {
			log.Debugf("open batch with max size %d not found, creating new one", openBatchMaxSize)
			newID, err := s.newID()
			if err != nil {
				return fmt.Errorf("create open batch id: %s", err)
			}
			newBatchID := broker.StorageDealID(newID)
			if err := queries.CreateOpenBatch(ctx, newBatchID); err != nil {
				return fmt.Errorf("creating open batch: %s", err)
			}
			ob = db.Batch{
				BatchID: broker.StorageDealID(newBatchID),
				Status:  db.BatchStatusOpen,
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
			}
			if _, err := queries.MoveBatchToStatus(ctx, mbtsParams); err != nil {
				return fmt.Errorf("move batch to status: %s", err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("executing tx: %s", err)
	}

	log.Debugf("opID %s, storage-request %s, data-cid %s included in batch %s", opID, srID, dataCid, ob.BatchID)

	return nil
}

// MoveBatchToStatus moves an executing unprepared job to a new status.
func (s *Store) MoveBatchToStatus(
	ctx context.Context,
	batchID broker.StorageDealID,
	status BatchStatus) error {
	dbStatus, err := statusToDB(status)
	if err != nil {
		return fmt.Errorf("casting status to db type: %s", err)
	}
	params := db.MoveBatchToStatusParams{
		BatchID: batchID,
		Status:  dbStatus,
	}
	count, err := s.db.MoveBatchToStatus(ctx, params)
	if err != nil {
		return fmt.Errorf("move batch to status: %s", err)
	}
	if count != 1 {
		return fmt.Errorf("unexpected update count, got: %d, expected: 1", count)
	}

	return nil
}

// GetNextReadyBatch returns the next ready batch to be processed batch and changes the
// status to Executing.
// The caller is responsible for updating the status later to Ready on error, or Done on success.
func (s *Store) GetNextReadyBatch(ctx context.Context) (broker.StorageDealID, int64, []StorageRequest, bool, error) {
	rb, err := s.db.GetNextReadyBatch(ctx)
	if err == sql.ErrNoRows {
		return "", 0, nil, false, nil
	}
	if err != nil {
		return "", 0, nil, false, fmt.Errorf("db get next ready: %s", err)
	}

	srs, err := s.db.GetStorageRequestFromBatch(ctx, rb.BatchID)
	if err != nil {
		return "", 0, nil, false, nil
	}

	return rb.BatchID, rb.TotalSize, srs, true, nil
}

// Stats provides stats for metrics.
type Stats struct {
	PendingStorageRequestsCount int64
	OpenBatchBytes              int64

	OpenBatchCount int64

	DoneBatchCount int64
	DoneBatchBytes int64
}

// GetStats return stats about batches.
func (s *Store) GetStats(ctx context.Context) (Stats, error) {
	pendStats, err := s.db.OpenBatchStats(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("open batch stats stats: %s", err)
	}

	doneStats, err := s.db.DoneBatchStats(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("done batch stats: %s", err)
	}

	return Stats{
		PendingStorageRequestsCount: pendStats.SrsCount,
		OpenBatchBytes:              pendStats.BatchesBytes,
		OpenBatchCount:              pendStats.BatchesCount,
		DoneBatchCount:              doneStats.DoneBatchCount,
		DoneBatchBytes:              doneStats.DoneBatchBytes,
	}, nil
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
	}

	return "", fmt.Errorf("unknown status: %#v", status)
}
