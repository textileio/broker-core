package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/textileio/broker-core/cmd/brokerd/store/internal/db"
)

// UnpinType is the type of an UnpinJob.
type UnpinType int

const (
	// UnpinTypeBatch is the type of batch unpins.
	UnpinTypeBatch UnpinType = iota
	// UnpinTypeData is the type of data unpins.
	UnpinTypeData

	stuckSeconds = int64(300)
)

// UnpinJob describes a job to unpin a Cid.
type UnpinJob db.UnpinJob

// UnpinJobGetNext returns the next pending unpin job to execute.
func (s *Store) UnpinJobGetNext(ctx context.Context) (UnpinJob, bool, error) {
	job, err := s.db.NextUnpinJob(ctx, stuckSeconds)
	if err == sql.ErrNoRows {
		return UnpinJob{}, false, nil
	}
	if err != nil {
		return UnpinJob{}, false, err
	}
	if int64(time.Since(job.ReadyAt).Seconds()) > stuckSeconds {
		log.Warnf("re-executing unpinning job %s", job.ID)
	}
	return UnpinJob(job), true, nil
}

// DeleteExecuting removes an executing unpin job.
func (s *Store) DeleteExecuting(ctx context.Context, id string) error {
	return s.db.DeleteExecutingUnpinJob(ctx, id)
}

// UnpinJobMoveToPending moves an unpin job to pending status.
func (s *Store) UnpinJobMoveToPending(ctx context.Context, id string, delay time.Duration) error {
	return s.db.UnpinJobToPending(ctx, db.UnpinJobToPendingParams{ID: id, ReadyAt: time.Now().Add(delay)})
}
