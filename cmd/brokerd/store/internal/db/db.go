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
	if q.batchUpdateBrokerRequestsStmt, err = db.PrepareContext(ctx, batchUpdateBrokerRequests); err != nil {
		return nil, fmt.Errorf("error preparing query BatchUpdateBrokerRequests: %w", err)
	}
	if q.createBrokerRequestStmt, err = db.PrepareContext(ctx, createBrokerRequest); err != nil {
		return nil, fmt.Errorf("error preparing query CreateBrokerRequest: %w", err)
	}
	if q.createMinerDealStmt, err = db.PrepareContext(ctx, createMinerDeal); err != nil {
		return nil, fmt.Errorf("error preparing query CreateMinerDeal: %w", err)
	}
	if q.createStorageDealStmt, err = db.PrepareContext(ctx, createStorageDeal); err != nil {
		return nil, fmt.Errorf("error preparing query CreateStorageDeal: %w", err)
	}
	if q.createUnpinJobStmt, err = db.PrepareContext(ctx, createUnpinJob); err != nil {
		return nil, fmt.Errorf("error preparing query CreateUnpinJob: %w", err)
	}
	if q.deleteExecutingUnpinJobStmt, err = db.PrepareContext(ctx, deleteExecutingUnpinJob); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteExecutingUnpinJob: %w", err)
	}
	if q.getBrokerRequestStmt, err = db.PrepareContext(ctx, getBrokerRequest); err != nil {
		return nil, fmt.Errorf("error preparing query GetBrokerRequest: %w", err)
	}
	if q.getBrokerRequestIDsStmt, err = db.PrepareContext(ctx, getBrokerRequestIDs); err != nil {
		return nil, fmt.Errorf("error preparing query GetBrokerRequestIDs: %w", err)
	}
	if q.getBrokerRequestsStmt, err = db.PrepareContext(ctx, getBrokerRequests); err != nil {
		return nil, fmt.Errorf("error preparing query GetBrokerRequests: %w", err)
	}
	if q.getMinerDealsStmt, err = db.PrepareContext(ctx, getMinerDeals); err != nil {
		return nil, fmt.Errorf("error preparing query GetMinerDeals: %w", err)
	}
	if q.getStorageDealStmt, err = db.PrepareContext(ctx, getStorageDeal); err != nil {
		return nil, fmt.Errorf("error preparing query GetStorageDeal: %w", err)
	}
	if q.nextUnpinJobStmt, err = db.PrepareContext(ctx, nextUnpinJob); err != nil {
		return nil, fmt.Errorf("error preparing query NextUnpinJob: %w", err)
	}
	if q.rebatchBrokerRequestsStmt, err = db.PrepareContext(ctx, rebatchBrokerRequests); err != nil {
		return nil, fmt.Errorf("error preparing query RebatchBrokerRequests: %w", err)
	}
	if q.unpinJobToPendingStmt, err = db.PrepareContext(ctx, unpinJobToPending); err != nil {
		return nil, fmt.Errorf("error preparing query UnpinJobToPending: %w", err)
	}
	if q.updateBrokerRequestsStatusStmt, err = db.PrepareContext(ctx, updateBrokerRequestsStatus); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateBrokerRequestsStatus: %w", err)
	}
	if q.updateMinerDealsStmt, err = db.PrepareContext(ctx, updateMinerDeals); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateMinerDeals: %w", err)
	}
	if q.updateStorageDealStmt, err = db.PrepareContext(ctx, updateStorageDeal); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateStorageDeal: %w", err)
	}
	if q.updateStorageDealStatusStmt, err = db.PrepareContext(ctx, updateStorageDealStatus); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateStorageDealStatus: %w", err)
	}
	if q.updateStorageDealStatusAndErrorStmt, err = db.PrepareContext(ctx, updateStorageDealStatusAndError); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateStorageDealStatusAndError: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.batchUpdateBrokerRequestsStmt != nil {
		if cerr := q.batchUpdateBrokerRequestsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing batchUpdateBrokerRequestsStmt: %w", cerr)
		}
	}
	if q.createBrokerRequestStmt != nil {
		if cerr := q.createBrokerRequestStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createBrokerRequestStmt: %w", cerr)
		}
	}
	if q.createMinerDealStmt != nil {
		if cerr := q.createMinerDealStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createMinerDealStmt: %w", cerr)
		}
	}
	if q.createStorageDealStmt != nil {
		if cerr := q.createStorageDealStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createStorageDealStmt: %w", cerr)
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
	if q.getBrokerRequestStmt != nil {
		if cerr := q.getBrokerRequestStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getBrokerRequestStmt: %w", cerr)
		}
	}
	if q.getBrokerRequestIDsStmt != nil {
		if cerr := q.getBrokerRequestIDsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getBrokerRequestIDsStmt: %w", cerr)
		}
	}
	if q.getBrokerRequestsStmt != nil {
		if cerr := q.getBrokerRequestsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getBrokerRequestsStmt: %w", cerr)
		}
	}
	if q.getMinerDealsStmt != nil {
		if cerr := q.getMinerDealsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getMinerDealsStmt: %w", cerr)
		}
	}
	if q.getStorageDealStmt != nil {
		if cerr := q.getStorageDealStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getStorageDealStmt: %w", cerr)
		}
	}
	if q.nextUnpinJobStmt != nil {
		if cerr := q.nextUnpinJobStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing nextUnpinJobStmt: %w", cerr)
		}
	}
	if q.rebatchBrokerRequestsStmt != nil {
		if cerr := q.rebatchBrokerRequestsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing rebatchBrokerRequestsStmt: %w", cerr)
		}
	}
	if q.unpinJobToPendingStmt != nil {
		if cerr := q.unpinJobToPendingStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing unpinJobToPendingStmt: %w", cerr)
		}
	}
	if q.updateBrokerRequestsStatusStmt != nil {
		if cerr := q.updateBrokerRequestsStatusStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateBrokerRequestsStatusStmt: %w", cerr)
		}
	}
	if q.updateMinerDealsStmt != nil {
		if cerr := q.updateMinerDealsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateMinerDealsStmt: %w", cerr)
		}
	}
	if q.updateStorageDealStmt != nil {
		if cerr := q.updateStorageDealStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateStorageDealStmt: %w", cerr)
		}
	}
	if q.updateStorageDealStatusStmt != nil {
		if cerr := q.updateStorageDealStatusStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateStorageDealStatusStmt: %w", cerr)
		}
	}
	if q.updateStorageDealStatusAndErrorStmt != nil {
		if cerr := q.updateStorageDealStatusAndErrorStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateStorageDealStatusAndErrorStmt: %w", cerr)
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
	db                                  DBTX
	tx                                  *sql.Tx
	batchUpdateBrokerRequestsStmt       *sql.Stmt
	createBrokerRequestStmt             *sql.Stmt
	createMinerDealStmt                 *sql.Stmt
	createStorageDealStmt               *sql.Stmt
	createUnpinJobStmt                  *sql.Stmt
	deleteExecutingUnpinJobStmt         *sql.Stmt
	getBrokerRequestStmt                *sql.Stmt
	getBrokerRequestIDsStmt             *sql.Stmt
	getBrokerRequestsStmt               *sql.Stmt
	getMinerDealsStmt                   *sql.Stmt
	getStorageDealStmt                  *sql.Stmt
	nextUnpinJobStmt                    *sql.Stmt
	rebatchBrokerRequestsStmt           *sql.Stmt
	unpinJobToPendingStmt               *sql.Stmt
	updateBrokerRequestsStatusStmt      *sql.Stmt
	updateMinerDealsStmt                *sql.Stmt
	updateStorageDealStmt               *sql.Stmt
	updateStorageDealStatusStmt         *sql.Stmt
	updateStorageDealStatusAndErrorStmt *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                                  tx,
		tx:                                  tx,
		batchUpdateBrokerRequestsStmt:       q.batchUpdateBrokerRequestsStmt,
		createBrokerRequestStmt:             q.createBrokerRequestStmt,
		createMinerDealStmt:                 q.createMinerDealStmt,
		createStorageDealStmt:               q.createStorageDealStmt,
		createUnpinJobStmt:                  q.createUnpinJobStmt,
		deleteExecutingUnpinJobStmt:         q.deleteExecutingUnpinJobStmt,
		getBrokerRequestStmt:                q.getBrokerRequestStmt,
		getBrokerRequestIDsStmt:             q.getBrokerRequestIDsStmt,
		getBrokerRequestsStmt:               q.getBrokerRequestsStmt,
		getMinerDealsStmt:                   q.getMinerDealsStmt,
		getStorageDealStmt:                  q.getStorageDealStmt,
		nextUnpinJobStmt:                    q.nextUnpinJobStmt,
		rebatchBrokerRequestsStmt:           q.rebatchBrokerRequestsStmt,
		unpinJobToPendingStmt:               q.unpinJobToPendingStmt,
		updateBrokerRequestsStatusStmt:      q.updateBrokerRequestsStatusStmt,
		updateMinerDealsStmt:                q.updateMinerDealsStmt,
		updateStorageDealStmt:               q.updateStorageDealStmt,
		updateStorageDealStatusStmt:         q.updateStorageDealStatusStmt,
		updateStorageDealStatusAndErrorStmt: q.updateStorageDealStatusAndErrorStmt,
	}
}
