// Code generated by sqlc. DO NOT EDIT.

package db

import (
	"context"
	"database/sql"
	"fmt"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.batchUpdateStorageRequestsStmt, err = db.PrepareContext(ctx, batchUpdateStorageRequests); err != nil {
		return nil, fmt.Errorf("error preparing query BatchUpdateStorageRequests: %w", err)
	}
	if q.createBatchStmt, err = db.PrepareContext(ctx, createBatch); err != nil {
		return nil, fmt.Errorf("error preparing query CreateBatch: %w", err)
	}
	if q.createBatchTagStmt, err = db.PrepareContext(ctx, createBatchTag); err != nil {
		return nil, fmt.Errorf("error preparing query CreateBatchTag: %w", err)
	}
	if q.createDealStmt, err = db.PrepareContext(ctx, createDeal); err != nil {
		return nil, fmt.Errorf("error preparing query CreateDeal: %w", err)
	}
	if q.createOperationStmt, err = db.PrepareContext(ctx, createOperation); err != nil {
		return nil, fmt.Errorf("error preparing query CreateOperation: %w", err)
	}
	if q.createStorageRequestStmt, err = db.PrepareContext(ctx, createStorageRequest); err != nil {
		return nil, fmt.Errorf("error preparing query CreateStorageRequest: %w", err)
	}
	if q.createUnpinJobStmt, err = db.PrepareContext(ctx, createUnpinJob); err != nil {
		return nil, fmt.Errorf("error preparing query CreateUnpinJob: %w", err)
	}
	if q.deleteExecutingUnpinJobStmt, err = db.PrepareContext(ctx, deleteExecutingUnpinJob); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteExecutingUnpinJob: %w", err)
	}
	if q.getBatchStmt, err = db.PrepareContext(ctx, getBatch); err != nil {
		return nil, fmt.Errorf("error preparing query GetBatch: %w", err)
	}
	if q.getBatchTagsStmt, err = db.PrepareContext(ctx, getBatchTags); err != nil {
		return nil, fmt.Errorf("error preparing query GetBatchTags: %w", err)
	}
	if q.getDealsStmt, err = db.PrepareContext(ctx, getDeals); err != nil {
		return nil, fmt.Errorf("error preparing query GetDeals: %w", err)
	}
	if q.getStorageRequestStmt, err = db.PrepareContext(ctx, getStorageRequest); err != nil {
		return nil, fmt.Errorf("error preparing query GetStorageRequest: %w", err)
	}
	if q.getStorageRequestIDsStmt, err = db.PrepareContext(ctx, getStorageRequestIDs); err != nil {
		return nil, fmt.Errorf("error preparing query GetStorageRequestIDs: %w", err)
	}
	if q.getStorageRequestsStmt, err = db.PrepareContext(ctx, getStorageRequests); err != nil {
		return nil, fmt.Errorf("error preparing query GetStorageRequests: %w", err)
	}
	if q.nextUnpinJobStmt, err = db.PrepareContext(ctx, nextUnpinJob); err != nil {
		return nil, fmt.Errorf("error preparing query NextUnpinJob: %w", err)
	}
	if q.rebatchStorageRequestsStmt, err = db.PrepareContext(ctx, rebatchStorageRequests); err != nil {
		return nil, fmt.Errorf("error preparing query RebatchStorageRequests: %w", err)
	}
	if q.unpinJobToPendingStmt, err = db.PrepareContext(ctx, unpinJobToPending); err != nil {
		return nil, fmt.Errorf("error preparing query UnpinJobToPending: %w", err)
	}
	if q.updateBatchStmt, err = db.PrepareContext(ctx, updateBatch); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateBatch: %w", err)
	}
	if q.updateBatchStatusStmt, err = db.PrepareContext(ctx, updateBatchStatus); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateBatchStatus: %w", err)
	}
	if q.updateBatchStatusAndErrorStmt, err = db.PrepareContext(ctx, updateBatchStatusAndError); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateBatchStatusAndError: %w", err)
	}
	if q.updateDealsStmt, err = db.PrepareContext(ctx, updateDeals); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateDeals: %w", err)
	}
	if q.updateStorageRequestsStatusStmt, err = db.PrepareContext(ctx, updateStorageRequestsStatus); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateStorageRequestsStatus: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.batchUpdateStorageRequestsStmt != nil {
		if cerr := q.batchUpdateStorageRequestsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing batchUpdateStorageRequestsStmt: %w", cerr)
		}
	}
	if q.createBatchStmt != nil {
		if cerr := q.createBatchStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createBatchStmt: %w", cerr)
		}
	}
	if q.createBatchTagStmt != nil {
		if cerr := q.createBatchTagStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createBatchTagStmt: %w", cerr)
		}
	}
	if q.createDealStmt != nil {
		if cerr := q.createDealStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createDealStmt: %w", cerr)
		}
	}
	if q.createOperationStmt != nil {
		if cerr := q.createOperationStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createOperationStmt: %w", cerr)
		}
	}
	if q.createStorageRequestStmt != nil {
		if cerr := q.createStorageRequestStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createStorageRequestStmt: %w", cerr)
		}
	}
	if q.createUnpinJobStmt != nil {
		if cerr := q.createUnpinJobStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createUnpinJobStmt: %w", cerr)
		}
	}
	if q.deleteExecutingUnpinJobStmt != nil {
		if cerr := q.deleteExecutingUnpinJobStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteExecutingUnpinJobStmt: %w", cerr)
		}
	}
	if q.getBatchStmt != nil {
		if cerr := q.getBatchStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getBatchStmt: %w", cerr)
		}
	}
	if q.getBatchTagsStmt != nil {
		if cerr := q.getBatchTagsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getBatchTagsStmt: %w", cerr)
		}
	}
	if q.getDealsStmt != nil {
		if cerr := q.getDealsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getDealsStmt: %w", cerr)
		}
	}
	if q.getStorageRequestStmt != nil {
		if cerr := q.getStorageRequestStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getStorageRequestStmt: %w", cerr)
		}
	}
	if q.getStorageRequestIDsStmt != nil {
		if cerr := q.getStorageRequestIDsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getStorageRequestIDsStmt: %w", cerr)
		}
	}
	if q.getStorageRequestsStmt != nil {
		if cerr := q.getStorageRequestsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getStorageRequestsStmt: %w", cerr)
		}
	}
	if q.nextUnpinJobStmt != nil {
		if cerr := q.nextUnpinJobStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing nextUnpinJobStmt: %w", cerr)
		}
	}
	if q.rebatchStorageRequestsStmt != nil {
		if cerr := q.rebatchStorageRequestsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing rebatchStorageRequestsStmt: %w", cerr)
		}
	}
	if q.unpinJobToPendingStmt != nil {
		if cerr := q.unpinJobToPendingStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing unpinJobToPendingStmt: %w", cerr)
		}
	}
	if q.updateBatchStmt != nil {
		if cerr := q.updateBatchStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateBatchStmt: %w", cerr)
		}
	}
	if q.updateBatchStatusStmt != nil {
		if cerr := q.updateBatchStatusStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateBatchStatusStmt: %w", cerr)
		}
	}
	if q.updateBatchStatusAndErrorStmt != nil {
		if cerr := q.updateBatchStatusAndErrorStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateBatchStatusAndErrorStmt: %w", cerr)
		}
	}
	if q.updateDealsStmt != nil {
		if cerr := q.updateDealsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateDealsStmt: %w", cerr)
		}
	}
	if q.updateStorageRequestsStatusStmt != nil {
		if cerr := q.updateStorageRequestsStatusStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateStorageRequestsStatusStmt: %w", cerr)
		}
	}
	return err
}

func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (sql.Result, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

type Queries struct {
	db                              DBTX
	tx                              *sql.Tx
	batchUpdateStorageRequestsStmt  *sql.Stmt
	createBatchStmt                 *sql.Stmt
	createBatchTagStmt              *sql.Stmt
	createDealStmt                  *sql.Stmt
	createOperationStmt             *sql.Stmt
	createStorageRequestStmt        *sql.Stmt
	createUnpinJobStmt              *sql.Stmt
	deleteExecutingUnpinJobStmt     *sql.Stmt
	getBatchStmt                    *sql.Stmt
	getBatchTagsStmt                *sql.Stmt
	getDealsStmt                    *sql.Stmt
	getStorageRequestStmt           *sql.Stmt
	getStorageRequestIDsStmt        *sql.Stmt
	getStorageRequestsStmt          *sql.Stmt
	nextUnpinJobStmt                *sql.Stmt
	rebatchStorageRequestsStmt      *sql.Stmt
	unpinJobToPendingStmt           *sql.Stmt
	updateBatchStmt                 *sql.Stmt
	updateBatchStatusStmt           *sql.Stmt
	updateBatchStatusAndErrorStmt   *sql.Stmt
	updateDealsStmt                 *sql.Stmt
	updateStorageRequestsStatusStmt *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                              tx,
		tx:                              tx,
		batchUpdateStorageRequestsStmt:  q.batchUpdateStorageRequestsStmt,
		createBatchStmt:                 q.createBatchStmt,
		createBatchTagStmt:              q.createBatchTagStmt,
		createDealStmt:                  q.createDealStmt,
		createOperationStmt:             q.createOperationStmt,
		createStorageRequestStmt:        q.createStorageRequestStmt,
		createUnpinJobStmt:              q.createUnpinJobStmt,
		deleteExecutingUnpinJobStmt:     q.deleteExecutingUnpinJobStmt,
		getBatchStmt:                    q.getBatchStmt,
		getBatchTagsStmt:                q.getBatchTagsStmt,
		getDealsStmt:                    q.getDealsStmt,
		getStorageRequestStmt:           q.getStorageRequestStmt,
		getStorageRequestIDsStmt:        q.getStorageRequestIDsStmt,
		getStorageRequestsStmt:          q.getStorageRequestsStmt,
		nextUnpinJobStmt:                q.nextUnpinJobStmt,
		rebatchStorageRequestsStmt:      q.rebatchStorageRequestsStmt,
		unpinJobToPendingStmt:           q.unpinJobToPendingStmt,
		updateBatchStmt:                 q.updateBatchStmt,
		updateBatchStatusStmt:           q.updateBatchStatusStmt,
		updateBatchStatusAndErrorStmt:   q.updateBatchStatusAndErrorStmt,
		updateDealsStmt:                 q.updateDealsStmt,
		updateStorageRequestsStatusStmt: q.updateStorageRequestsStatusStmt,
	}
}
