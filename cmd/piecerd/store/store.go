package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/ipfs/go-cid"
	"github.com/jackc/pgconn"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/piecerd/store/internal/db"
	"github.com/textileio/broker-core/cmd/piecerd/store/migrations"
	"github.com/textileio/broker-core/storeutil"
	logger "github.com/textileio/go-log/v2"
)

const (
	stuckEpochs = int64(7200)
)

var (
	// ErrBatchExists if the provided batch id already exists.
	ErrBatchExists = errors.New("batch-id already exists")

	log = logger.Logger("store")
)

// UnpreparedBatchStatus is the status of an unprepared batch.
type UnpreparedBatchStatus int

const (
	// StatusPending is an unprepared batch ready to prepared.
	StatusPending UnpreparedBatchStatus = iota
	// StatusExecuting is an unprepared batch being prepared.
	StatusExecuting
	// StatusDone is an unprepared batch already prepared.
	StatusDone
)

// UnpreparedBatch is a batch that is ready to be prepared.
type UnpreparedBatch struct {
	BatchID   broker.BatchID
	DataCid   cid.Cid
	ReadyAt   time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Store is a store for unprepared batches.
type Store struct {
	conn *sql.DB
	db   *db.Queries
}

// New returns a new Store.
func New(postgresURI string) (*Store, error) {
	as := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})
	conn, err := storeutil.MigrateAndConnectToDB(postgresURI, as)
	if err != nil {
		return nil, fmt.Errorf("initializing db connection: %s", err)
	}

	s := &Store{
		conn: conn,
		db:   db.New(conn),
	}

	return s, nil
}

// CreateUnpreparedBatch creates a new pending unprepared batch to be prepared.
func (s *Store) CreateUnpreparedBatch(ctx context.Context, sdID broker.BatchID, dataCid cid.Cid) error {
	if sdID == "" {
		return errors.New("batch-id is empty")
	}
	if !dataCid.Defined() {
		return errors.New("data-cid is undefined")
	}
	params := db.CreateUnpreparedBatchParams{
		BatchID: sdID,
		DataCid: dataCid.String(),
	}
	if err := s.db.CreateUnpreparedBatch(ctx, params); err != nil {
		if err, ok := err.(*pgconn.PgError); ok {
			if err.Code == "23505" {
				return ErrBatchExists
			}
		}
		return fmt.Errorf("db create unprepared batch: %s", err)
	}

	return nil
}

// GetNextPending returns the next pending batch to process and set the status to Executing.
// The caller is responsible for updating the status later to Pending on error, or Done on success.
func (s *Store) GetNextPending(ctx context.Context) (UnpreparedBatch, bool, error) {
	ub, err := s.db.GetNextPending(ctx, stuckEpochs)
	if err == sql.ErrNoRows {
		return UnpreparedBatch{}, false, nil
	}
	if err != nil {
		return UnpreparedBatch{}, false, fmt.Errorf("db get next pending: %s", err)
	}

	if int64(time.Since(ub.ReadyAt).Seconds()) > stuckEpochs {
		log.Warnf("re-executing stuck batch %s", ub.BatchID)
	}

	dataCid, err := cid.Decode(ub.DataCid)
	if err != nil {
		return UnpreparedBatch{}, false, fmt.Errorf("parsing cid %s: %s", ub.DataCid, err)
	}

	return UnpreparedBatch{
		BatchID:   ub.BatchID,
		DataCid:   dataCid,
		ReadyAt:   ub.ReadyAt,
		CreatedAt: ub.CreatedAt,
		UpdatedAt: ub.UpdatedAt,
	}, true, nil
}

// MoveToStatus moves an executing unprepared job to a new status.
func (s *Store) MoveToStatus(
	ctx context.Context,
	sdID broker.BatchID,
	delay time.Duration,
	status UnpreparedBatchStatus) error {
	dbStatus, err := statusToDB(status)
	if err != nil {
		return fmt.Errorf("casting status to db type: %s", err)
	}
	params := db.MoveToStatusParams{
		BatchID: sdID,
		ReadyAt: time.Now().Add(delay),
		Status:  dbStatus,
	}
	count, err := s.db.MoveToStatus(ctx, params)
	if err != nil {
		return fmt.Errorf("delete from database: %s", err)
	}
	if count != 1 {
		return fmt.Errorf("unexpected update count, got: %d, expected: 1", count)
	}

	return nil
}

// Close closes the store.
func (s *Store) Close() error {
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("closing sql connection: %s", err)
	}
	return nil
}

func statusToDB(status UnpreparedBatchStatus) (db.UnpreparedBatchStatus, error) {
	switch status {
	case StatusPending:
		return db.UnpreparedBatchStatusPending, nil
	case StatusExecuting:
		return db.UnpreparedBatchStatusExecuting, nil
	case StatusDone:
		return db.UnpreparedBatchStatusDone, nil
	}

	return "", fmt.Errorf("unknown status: %#v", status)
}
