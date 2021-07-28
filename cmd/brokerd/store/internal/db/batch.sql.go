// Code generated by sqlc. DO NOT EDIT.
// source: batch.sql

package db

import (
	"context"

	"github.com/textileio/broker-core/broker"
)

const createBatch = `-- name: CreateBatch :exec
INSERT INTO batches(
    id,
    status,
    rep_factor,
    deal_duration,
    car_url,
    car_ipfs_cid,
    car_ipfs_addrs,
    disallow_rebatching,
    fil_epoch_deadline,
    error,
    payload_cid,
    piece_cid,
    piece_size,
    origin
    ) VALUES (
      $1,
      $2,
      $3,
      $4,
      $5,
      $6,
      $7,
      $8,
      $9,
      $10,
      $11,
      $12,
      $13,
      $14)
`

type CreateBatchParams struct {
	ID                 broker.BatchID     `json:"id"`
	Status             broker.BatchStatus `json:"status"`
	RepFactor          int                `json:"repFactor"`
	DealDuration       int                `json:"dealDuration"`
	CarUrl             string             `json:"carUrl"`
	CarIpfsCid         string             `json:"carIpfsCid"`
	CarIpfsAddrs       string             `json:"carIpfsAddrs"`
	DisallowRebatching bool               `json:"disallowRebatching"`
	FilEpochDeadline   uint64             `json:"filEpochDeadline"`
	Error              string             `json:"error"`
	PayloadCid         string             `json:"payloadCid"`
	PieceCid           string             `json:"pieceCid"`
	PieceSize          uint64             `json:"pieceSize"`
	Origin             string             `json:"origin"`
}

func (q *Queries) CreateBatch(ctx context.Context, arg CreateBatchParams) error {
	_, err := q.exec(ctx, q.createBatchStmt, createBatch,
		arg.ID,
		arg.Status,
		arg.RepFactor,
		arg.DealDuration,
		arg.CarUrl,
		arg.CarIpfsCid,
		arg.CarIpfsAddrs,
		arg.DisallowRebatching,
		arg.FilEpochDeadline,
		arg.Error,
		arg.PayloadCid,
		arg.PieceCid,
		arg.PieceSize,
		arg.Origin,
	)
	return err
}

const createBatchTag = `-- name: CreateBatchTag :exec
INSERT INTO batch_tags (batch_id,key,value) VALUES ($1,$2,$3)
`

type CreateBatchTagParams struct {
	BatchID string `json:"batchID"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

func (q *Queries) CreateBatchTag(ctx context.Context, arg CreateBatchTagParams) error {
	_, err := q.exec(ctx, q.createBatchTagStmt, createBatchTag, arg.BatchID, arg.Key, arg.Value)
	return err
}

const getBatch = `-- name: GetBatch :one
SELECT id, status, rep_factor, deal_duration, payload_cid, piece_cid, piece_size, car_url, car_ipfs_cid, car_ipfs_addrs, disallow_rebatching, fil_epoch_deadline, error, origin, created_at, updated_at FROM batches
WHERE id = $1
`

func (q *Queries) GetBatch(ctx context.Context, id broker.BatchID) (Batch, error) {
	row := q.queryRow(ctx, q.getBatchStmt, getBatch, id)
	var i Batch
	err := row.Scan(
		&i.ID,
		&i.Status,
		&i.RepFactor,
		&i.DealDuration,
		&i.PayloadCid,
		&i.PieceCid,
		&i.PieceSize,
		&i.CarUrl,
		&i.CarIpfsCid,
		&i.CarIpfsAddrs,
		&i.DisallowRebatching,
		&i.FilEpochDeadline,
		&i.Error,
		&i.Origin,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateBatch = `-- name: UpdateBatch :exec
UPDATE batches
SET status = $2,
    piece_cid = $3,
    piece_size = $4,
    updated_at = CURRENT_TIMESTAMP
    WHERE id = $1
`

type UpdateBatchParams struct {
	ID        broker.BatchID     `json:"id"`
	Status    broker.BatchStatus `json:"status"`
	PieceCid  string             `json:"pieceCid"`
	PieceSize uint64             `json:"pieceSize"`
}

func (q *Queries) UpdateBatch(ctx context.Context, arg UpdateBatchParams) error {
	_, err := q.exec(ctx, q.updateBatchStmt, updateBatch,
		arg.ID,
		arg.Status,
		arg.PieceCid,
		arg.PieceSize,
	)
	return err
}

const updateBatchStatus = `-- name: UpdateBatchStatus :exec
UPDATE batches
SET status = $2,
    updated_at = CURRENT_TIMESTAMP
    WHERE id = $1
`

type UpdateBatchStatusParams struct {
	ID     broker.BatchID     `json:"id"`
	Status broker.BatchStatus `json:"status"`
}

func (q *Queries) UpdateBatchStatus(ctx context.Context, arg UpdateBatchStatusParams) error {
	_, err := q.exec(ctx, q.updateBatchStatusStmt, updateBatchStatus, arg.ID, arg.Status)
	return err
}

const updateBatchStatusAndError = `-- name: UpdateBatchStatusAndError :exec
UPDATE batches
SET status = $2,
    error = $3,
    updated_at = CURRENT_TIMESTAMP
    WHERE id = $1
`

type UpdateBatchStatusAndErrorParams struct {
	ID     broker.BatchID     `json:"id"`
	Status broker.BatchStatus `json:"status"`
	Error  string             `json:"error"`
}

func (q *Queries) UpdateBatchStatusAndError(ctx context.Context, arg UpdateBatchStatusAndErrorParams) error {
	_, err := q.exec(ctx, q.updateBatchStatusAndErrorStmt, updateBatchStatusAndError, arg.ID, arg.Status, arg.Error)
	return err
}
