// Code generated by sqlc. DO NOT EDIT.
// source: deal.sql

package db

import (
	"context"

	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
)

const createDeal = `-- name: CreateDeal :exec
INSERT INTO deals(
    batch_id,
    auction_id,
    bid_id,
    storage_provider_id,
    deal_id,
    deal_expiration,
    error_cause
    ) VALUES(
      $1,
      $2,
      $3,
      $4,
      $5,
      $6,
      $7
      )
`

type CreateDealParams struct {
	BatchID           broker.BatchID `json:"batchID"`
	AuctionID         auction.ID     `json:"auctionID"`
	BidID             auction.BidID  `json:"bidID"`
	StorageProviderID string         `json:"storageProviderID"`
	DealID            int64          `json:"dealID"`
	DealExpiration    uint64         `json:"dealExpiration"`
	ErrorCause        string         `json:"errorCause"`
}

func (q *Queries) CreateDeal(ctx context.Context, arg CreateDealParams) error {
	_, err := q.exec(ctx, q.createDealStmt, createDeal,
		arg.BatchID,
		arg.AuctionID,
		arg.BidID,
		arg.StorageProviderID,
		arg.DealID,
		arg.DealExpiration,
		arg.ErrorCause,
	)
	return err
}

const getDeals = `-- name: GetDeals :many
SELECT batch_id, auction_id, bid_id, storage_provider_id, deal_id, deal_expiration, error_cause, created_at, updated_at FROM deals WHERE batch_id = $1
`

func (q *Queries) GetDeals(ctx context.Context, batchID broker.BatchID) ([]Deal, error) {
	rows, err := q.query(ctx, q.getDealsStmt, getDeals, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Deal
	for rows.Next() {
		var i Deal
		if err := rows.Scan(
			&i.BatchID,
			&i.AuctionID,
			&i.BidID,
			&i.StorageProviderID,
			&i.DealID,
			&i.DealExpiration,
			&i.ErrorCause,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getExcludedStorageProviders = `-- name: GetExcludedStorageProviders :many
SELECT DISTINCT d.storage_provider_id
FROM deals d
INNER JOIN batches b ON b.id=d.batch_id
WHERE b.piece_cid=$1 AND
      b.origin=$2
`

type GetExcludedStorageProvidersParams struct {
	PieceCid string `json:"pieceCid"`
	Origin   string `json:"origin"`
}

func (q *Queries) GetExcludedStorageProviders(ctx context.Context, arg GetExcludedStorageProvidersParams) ([]string, error) {
	rows, err := q.query(ctx, q.getExcludedStorageProvidersStmt, getExcludedStorageProviders, arg.PieceCid, arg.Origin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var storage_provider_id string
		if err := rows.Scan(&storage_provider_id); err != nil {
			return nil, err
		}
		items = append(items, storage_provider_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateDeals = `-- name: UpdateDeals :execrows
UPDATE deals
SET deal_id = $3,
    deal_expiration = $4,
    error_cause = $5,
    updated_at=CURRENT_TIMESTAMP
WHERE batch_id = $1 AND storage_provider_id = $2
`

type UpdateDealsParams struct {
	BatchID           broker.BatchID `json:"batchID"`
	StorageProviderID string         `json:"storageProviderID"`
	DealID            int64          `json:"dealID"`
	DealExpiration    uint64         `json:"dealExpiration"`
	ErrorCause        string         `json:"errorCause"`
}

func (q *Queries) UpdateDeals(ctx context.Context, arg UpdateDealsParams) (int64, error) {
	result, err := q.exec(ctx, q.updateDealsStmt, updateDeals,
		arg.BatchID,
		arg.StorageProviderID,
		arg.DealID,
		arg.DealExpiration,
		arg.ErrorCause,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
