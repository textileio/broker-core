package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/piecerd/store/internal/db"
	"github.com/textileio/broker-core/cmd/piecerd/store/migrations"
	"github.com/textileio/broker-core/storeutil"
	logger "github.com/textileio/go-log/v2"
)

var (
	log = logger.Logger("store")
)

type status int

const (
	statusPending status = iota
	statusExecuting
)

type UnpreparedBatch struct {
	StorageDealID broker.StorageDealID
	DataCid       cid.Cid
	ReadyAt       time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
func (s *Store) CreateUnpreparedBatch(ctx context.Context, sdID broker.StorageDealID, dataCid cid.Cid) error {
	if sdID == "" {
		return errors.New("storage-deal-id is empty")
	}
	if !dataCid.Defined() {
		return errors.New("data-cid is undefined")
	}
	params := db.CreateUnpreparedBatchParams{
		StorageDealID: sdID,
		Status:        int16(statusPending),
		DataCid:       dataCid.String(),
	}
	if err := s.db.CreateUnpreparedBatch(ctx, params); err != nil {
		return fmt.Errorf("db create unprepared batch: %s", err)
	}

	return nil
}

// GetNextPending returns the next pending batch to process and set the status to Executing.
// The caller is responsible for updating the status later to Pending on error, or deleting
// the record on success.
func (s *Store) GetNextPending(ctx context.Context) (UnpreparedBatch, bool, error) {
	ub, err := s.db.GetNextPending(ctx)
	if err == sql.ErrNoRows {
		return UnpreparedBatch{}, false, nil
	}
	if err != nil {
		return UnpreparedBatch{}, false, fmt.Errorf("db get next pending: %s", err)
	}

	dataCid, err := cid.Decode(ub.DataCid)
	if err != nil {
		return UnpreparedBatch{}, false, fmt.Errorf("parsing cid %s: %s", ub.DataCid, err)
	}

	return UnpreparedBatch{
		StorageDealID: ub.StorageDealID,
		DataCid:       dataCid,
		ReadyAt:       ub.ReadyAt,
		CreatedAt:     ub.CreatedAt,
		UpdatedAt:     ub.UpdatedAt,
	}, true, nil
}

// DeleteUnpreparedBatch deletes an Executing unprepared batch.
func (s *Store) DeleteUnpreparedBatch(ctx context.Context, sdID broker.StorageDealID) error {
	count, err := s.db.DeleteUnpreparedBatch(ctx, sdID)
	if err != nil {
		return fmt.Errorf("delete from database: %s", err)
	}
	if count != 1 {
		return fmt.Errorf("unexpected update count, got: %d, expected: 1", count)
	}

	return nil
}

// MoveToPending moves an executing unprepared job to pending status.
func (s *Store) MoveToPending(ctx context.Context, sdID broker.StorageDealID, delay time.Duration) error {
	params := db.MoveToPendingParams{
		StorageDealID: sdID,
		ReadyAt:       time.Now().Add(delay),
	}
	count, err := s.db.MoveToPending(ctx, params)
	if err != nil {
		return fmt.Errorf("delete from database: %s", err)
	}
	if count != 1 {
		return fmt.Errorf("unexpected update count, got: %d, expected: 1", count)
	}

	return nil
}
