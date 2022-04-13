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
	if q.createAuctionStmt, err = db.PrepareContext(ctx, createAuction); err != nil {
		return nil, fmt.Errorf("error preparing query CreateAuction: %w", err)
	}
	if q.createBidStmt, err = db.PrepareContext(ctx, createBid); err != nil {
		return nil, fmt.Errorf("error preparing query CreateBid: %w", err)
	}
	if q.createBidEventStmt, err = db.PrepareContext(ctx, createBidEvent); err != nil {
		return nil, fmt.Errorf("error preparing query CreateBidEvent: %w", err)
	}
	if q.createOrUpdateStorageProviderStmt, err = db.PrepareContext(ctx, createOrUpdateStorageProvider); err != nil {
		return nil, fmt.Errorf("error preparing query CreateOrUpdateStorageProvider: %w", err)
	}
	if q.getAuctionStmt, err = db.PrepareContext(ctx, getAuction); err != nil {
		return nil, fmt.Errorf("error preparing query GetAuction: %w", err)
	}
	if q.getAuctionBidsStmt, err = db.PrepareContext(ctx, getAuctionBids); err != nil {
		return nil, fmt.Errorf("error preparing query GetAuctionBids: %w", err)
	}
	if q.getAuctionWinningBidsStmt, err = db.PrepareContext(ctx, getAuctionWinningBids); err != nil {
		return nil, fmt.Errorf("error preparing query GetAuctionWinningBids: %w", err)
	}
	if q.getNextReadyToExecuteStmt, err = db.PrepareContext(ctx, getNextReadyToExecute); err != nil {
		return nil, fmt.Errorf("error preparing query GetNextReadyToExecute: %w", err)
	}
	if q.getRecentWeekFailureRateStmt, err = db.PrepareContext(ctx, getRecentWeekFailureRate); err != nil {
		return nil, fmt.Errorf("error preparing query GetRecentWeekFailureRate: %w", err)
	}
	if q.getRecentWeekMaxOnChainSecondsStmt, err = db.PrepareContext(ctx, getRecentWeekMaxOnChainSeconds); err != nil {
		return nil, fmt.Errorf("error preparing query GetRecentWeekMaxOnChainSeconds: %w", err)
	}
	if q.getRecentWeekWinningRateStmt, err = db.PrepareContext(ctx, getRecentWeekWinningRate); err != nil {
		return nil, fmt.Errorf("error preparing query GetRecentWeekWinningRate: %w", err)
	}
	if q.setStorageProviderUnhealthyStmt, err = db.PrepareContext(ctx, setStorageProviderUnhealthy); err != nil {
		return nil, fmt.Errorf("error preparing query SetStorageProviderUnhealthy: %w", err)
	}
	if q.updateAuctionStatusAndErrorStmt, err = db.PrepareContext(ctx, updateAuctionStatusAndError); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateAuctionStatusAndError: %w", err)
	}
	if q.updateDealConfirmedAtStmt, err = db.PrepareContext(ctx, updateDealConfirmedAt); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateDealConfirmedAt: %w", err)
	}
	if q.updateDealFailedAtStmt, err = db.PrepareContext(ctx, updateDealFailedAt); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateDealFailedAt: %w", err)
	}
	if q.updateProposalCidStmt, err = db.PrepareContext(ctx, updateProposalCid); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateProposalCid: %w", err)
	}
	if q.updateProposalCidDeliveryErrorStmt, err = db.PrepareContext(ctx, updateProposalCidDeliveryError); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateProposalCidDeliveryError: %w", err)
	}
	if q.updateWinningBidStmt, err = db.PrepareContext(ctx, updateWinningBid); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateWinningBid: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.createAuctionStmt != nil {
		if cerr := q.createAuctionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createAuctionStmt: %w", cerr)
		}
	}
	if q.createBidStmt != nil {
		if cerr := q.createBidStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createBidStmt: %w", cerr)
		}
	}
	if q.createBidEventStmt != nil {
		if cerr := q.createBidEventStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createBidEventStmt: %w", cerr)
		}
	}
	if q.createOrUpdateStorageProviderStmt != nil {
		if cerr := q.createOrUpdateStorageProviderStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createOrUpdateStorageProviderStmt: %w", cerr)
		}
	}
	if q.getAuctionStmt != nil {
		if cerr := q.getAuctionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getAuctionStmt: %w", cerr)
		}
	}
	if q.getAuctionBidsStmt != nil {
		if cerr := q.getAuctionBidsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getAuctionBidsStmt: %w", cerr)
		}
	}
	if q.getAuctionWinningBidsStmt != nil {
		if cerr := q.getAuctionWinningBidsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getAuctionWinningBidsStmt: %w", cerr)
		}
	}
	if q.getNextReadyToExecuteStmt != nil {
		if cerr := q.getNextReadyToExecuteStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNextReadyToExecuteStmt: %w", cerr)
		}
	}
	if q.getRecentWeekFailureRateStmt != nil {
		if cerr := q.getRecentWeekFailureRateStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getRecentWeekFailureRateStmt: %w", cerr)
		}
	}
	if q.getRecentWeekMaxOnChainSecondsStmt != nil {
		if cerr := q.getRecentWeekMaxOnChainSecondsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getRecentWeekMaxOnChainSecondsStmt: %w", cerr)
		}
	}
	if q.getRecentWeekWinningRateStmt != nil {
		if cerr := q.getRecentWeekWinningRateStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getRecentWeekWinningRateStmt: %w", cerr)
		}
	}
	if q.setStorageProviderUnhealthyStmt != nil {
		if cerr := q.setStorageProviderUnhealthyStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setStorageProviderUnhealthyStmt: %w", cerr)
		}
	}
	if q.updateAuctionStatusAndErrorStmt != nil {
		if cerr := q.updateAuctionStatusAndErrorStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateAuctionStatusAndErrorStmt: %w", cerr)
		}
	}
	if q.updateDealConfirmedAtStmt != nil {
		if cerr := q.updateDealConfirmedAtStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateDealConfirmedAtStmt: %w", cerr)
		}
	}
	if q.updateDealFailedAtStmt != nil {
		if cerr := q.updateDealFailedAtStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateDealFailedAtStmt: %w", cerr)
		}
	}
	if q.updateProposalCidStmt != nil {
		if cerr := q.updateProposalCidStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateProposalCidStmt: %w", cerr)
		}
	}
	if q.updateProposalCidDeliveryErrorStmt != nil {
		if cerr := q.updateProposalCidDeliveryErrorStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateProposalCidDeliveryErrorStmt: %w", cerr)
		}
	}
	if q.updateWinningBidStmt != nil {
		if cerr := q.updateWinningBidStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateWinningBidStmt: %w", cerr)
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
	db                                 DBTX
	tx                                 *sql.Tx
	createAuctionStmt                  *sql.Stmt
	createBidStmt                      *sql.Stmt
	createBidEventStmt                 *sql.Stmt
	createOrUpdateStorageProviderStmt  *sql.Stmt
	getAuctionStmt                     *sql.Stmt
	getAuctionBidsStmt                 *sql.Stmt
	getAuctionWinningBidsStmt          *sql.Stmt
	getNextReadyToExecuteStmt          *sql.Stmt
	getRecentWeekFailureRateStmt       *sql.Stmt
	getRecentWeekMaxOnChainSecondsStmt *sql.Stmt
	getRecentWeekWinningRateStmt       *sql.Stmt
	setStorageProviderUnhealthyStmt    *sql.Stmt
	updateAuctionStatusAndErrorStmt    *sql.Stmt
	updateDealConfirmedAtStmt          *sql.Stmt
	updateDealFailedAtStmt             *sql.Stmt
	updateProposalCidStmt              *sql.Stmt
	updateProposalCidDeliveryErrorStmt *sql.Stmt
	updateWinningBidStmt               *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                                 tx,
		tx:                                 tx,
		createAuctionStmt:                  q.createAuctionStmt,
		createBidStmt:                      q.createBidStmt,
		createBidEventStmt:                 q.createBidEventStmt,
		createOrUpdateStorageProviderStmt:  q.createOrUpdateStorageProviderStmt,
		getAuctionStmt:                     q.getAuctionStmt,
		getAuctionBidsStmt:                 q.getAuctionBidsStmt,
		getAuctionWinningBidsStmt:          q.getAuctionWinningBidsStmt,
		getNextReadyToExecuteStmt:          q.getNextReadyToExecuteStmt,
		getRecentWeekFailureRateStmt:       q.getRecentWeekFailureRateStmt,
		getRecentWeekMaxOnChainSecondsStmt: q.getRecentWeekMaxOnChainSecondsStmt,
		getRecentWeekWinningRateStmt:       q.getRecentWeekWinningRateStmt,
		setStorageProviderUnhealthyStmt:    q.setStorageProviderUnhealthyStmt,
		updateAuctionStatusAndErrorStmt:    q.updateAuctionStatusAndErrorStmt,
		updateDealConfirmedAtStmt:          q.updateDealConfirmedAtStmt,
		updateDealFailedAtStmt:             q.updateDealFailedAtStmt,
		updateProposalCidStmt:              q.updateProposalCidStmt,
		updateProposalCidDeliveryErrorStmt: q.updateProposalCidDeliveryErrorStmt,
		updateWinningBidStmt:               q.updateWinningBidStmt,
	}
}
