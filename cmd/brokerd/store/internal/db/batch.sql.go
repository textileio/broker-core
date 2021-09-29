// Code generated by sqlc. DO NOT EDIT.
// source: batch.sql

package db

import (
	"context"

	"github.com/lib/pq"
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
    origin,
    providers
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
      $14,
      $15)
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
	Providers          []string           `json:"providers"`
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
		pq.Array(arg.Providers),
	)
	return err
}

const createBatchManifest = `-- name: CreateBatchManifest :exec
INSERT INTO batch_manifests(
   batch_id,
   manifest
   ) VALUES (
     $1,
     $2)
`

type CreateBatchManifestParams struct {
	BatchID  string `json:"batchID"`
	Manifest []byte `json:"manifest"`
}

func (q *Queries) CreateBatchManifest(ctx context.Context, arg CreateBatchManifestParams) error {
	_, err := q.exec(ctx, q.createBatchManifestStmt, createBatchManifest, arg.BatchID, arg.Manifest)
	return err
}

const createBatchRemoteWallet = `-- name: CreateBatchRemoteWallet :exec
INSERT INTO batch_remote_wallet (
	batch_id,
	peer_id,
	auth_token,
	wallet_addr,
	multiaddrs
        ) VALUES (
	$1,
	$2,
	$3,
	$4,
	$5)
`

type CreateBatchRemoteWalletParams struct {
	BatchID    broker.BatchID `json:"batchID"`
	PeerID     string         `json:"peerID"`
	AuthToken  string         `json:"authToken"`
	WalletAddr string         `json:"walletAddr"`
	Multiaddrs []string       `json:"multiaddrs"`
}

func (q *Queries) CreateBatchRemoteWallet(ctx context.Context, arg CreateBatchRemoteWalletParams) error {
	_, err := q.exec(ctx, q.createBatchRemoteWalletStmt, createBatchRemoteWallet,
		arg.BatchID,
		arg.PeerID,
		arg.AuthToken,
		arg.WalletAddr,
		pq.Array(arg.Multiaddrs),
	)
	return err
}

const createBatchTag = `-- name: CreateBatchTag :exec
INSERT INTO batch_tags (batch_id,key,value) VALUES ($1,$2,$3)
`

type CreateBatchTagParams struct {
	BatchID broker.BatchID `json:"batchID"`
	Key     string         `json:"key"`
	Value   string         `json:"value"`
}

func (q *Queries) CreateBatchTag(ctx context.Context, arg CreateBatchTagParams) error {
	_, err := q.exec(ctx, q.createBatchTagStmt, createBatchTag, arg.BatchID, arg.Key, arg.Value)
	return err
}

const getBatch = `-- name: GetBatch :one
SELECT id, status, rep_factor, deal_duration, payload_cid, piece_cid, piece_size, car_url, car_ipfs_cid, car_ipfs_addrs, disallow_rebatching, fil_epoch_deadline, error, origin, created_at, updated_at, providers FROM batches
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
		pq.Array(&i.Providers),
	)
	return i, err
}

const getBatchManifest = `-- name: GetBatchManifest :one
SELECT batch_id, manifest FROM batch_manifests
WHERE batch_id=$1
`

func (q *Queries) GetBatchManifest(ctx context.Context, batchID string) (BatchManifest, error) {
	row := q.queryRow(ctx, q.getBatchManifestStmt, getBatchManifest, batchID)
	var i BatchManifest
	err := row.Scan(&i.BatchID, &i.Manifest)
	return i, err
}

const getBatchRemoteWallet = `-- name: GetBatchRemoteWallet :one
SELECT batch_id, peer_id, auth_token, wallet_addr, multiaddrs, created_at, updated_at FROM batch_remote_wallet
WHERE batch_id=$1
`

func (q *Queries) GetBatchRemoteWallet(ctx context.Context, batchID broker.BatchID) (BatchRemoteWallet, error) {
	row := q.queryRow(ctx, q.getBatchRemoteWalletStmt, getBatchRemoteWallet, batchID)
	var i BatchRemoteWallet
	err := row.Scan(
		&i.BatchID,
		&i.PeerID,
		&i.AuthToken,
		&i.WalletAddr,
		pq.Array(&i.Multiaddrs),
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getBatchTags = `-- name: GetBatchTags :many
SELECT batch_id, key, value, created_at FROM batch_tags
WHERE batch_id=$1
`

func (q *Queries) GetBatchTags(ctx context.Context, batchID broker.BatchID) ([]BatchTag, error) {
	rows, err := q.query(ctx, q.getBatchTagsStmt, getBatchTags, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []BatchTag
	for rows.Next() {
		var i BatchTag
		if err := rows.Scan(
			&i.BatchID,
			&i.Key,
			&i.Value,
			&i.CreatedAt,
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
