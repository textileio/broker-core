// Code generated by sqlc. DO NOT EDIT.
// source: auction_data.sql

package db

import (
	"context"

	"github.com/lib/pq"
	"github.com/textileio/broker-core/broker"
)

const createAuctionData = `-- name: CreateAuctionData :exec
INSERT INTO auction_data(
    batch_id,
    payload_cid,
    piece_cid,
    piece_size,
    duration
    ) VALUES (
      $1,
      $2,
      $3,
      $4,
      $5
      )
`

type CreateAuctionDataParams struct {
	BatchID    broker.BatchID `json:"batchID"`
	PayloadCid string         `json:"payloadCid"`
	PieceCid   string         `json:"pieceCid"`
	PieceSize  uint64         `json:"pieceSize"`
	Duration   uint64         `json:"duration"`
}

func (q *Queries) CreateAuctionData(ctx context.Context, arg CreateAuctionDataParams) error {
	_, err := q.exec(ctx, q.createAuctionDataStmt, createAuctionData,
		arg.BatchID,
		arg.PayloadCid,
		arg.PieceCid,
		arg.PieceSize,
		arg.Duration,
	)
	return err
}

const createRemoteWallet = `-- name: CreateRemoteWallet :exec
INSERT INTO remote_wallet(
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

type CreateRemoteWalletParams struct {
	BatchID    broker.BatchID `json:"batchID"`
	PeerID     string         `json:"peerID"`
	AuthToken  string         `json:"authToken"`
	WalletAddr string         `json:"walletAddr"`
	Multiaddrs []string       `json:"multiaddrs"`
}

func (q *Queries) CreateRemoteWallet(ctx context.Context, arg CreateRemoteWalletParams) error {
	_, err := q.exec(ctx, q.createRemoteWalletStmt, createRemoteWallet,
		arg.BatchID,
		arg.PeerID,
		arg.AuthToken,
		arg.WalletAddr,
		pq.Array(arg.Multiaddrs),
	)
	return err
}

const getAuctionData = `-- name: GetAuctionData :one
SELECT batch_id, payload_cid, piece_cid, piece_size, duration, created_at FROM auction_data
WHERE batch_id = $1
`

func (q *Queries) GetAuctionData(ctx context.Context, batchID broker.BatchID) (AuctionDatum, error) {
	row := q.queryRow(ctx, q.getAuctionDataStmt, getAuctionData, batchID)
	var i AuctionDatum
	err := row.Scan(
		&i.BatchID,
		&i.PayloadCid,
		&i.PieceCid,
		&i.PieceSize,
		&i.Duration,
		&i.CreatedAt,
	)
	return i, err
}

const getRemoteWallet = `-- name: GetRemoteWallet :one
SELECT peer_id, auth_token, wallet_addr, multiaddrs, created_at, updated_at, batch_id FROM remote_wallet
where batch_id = $1
`

func (q *Queries) GetRemoteWallet(ctx context.Context, batchID broker.BatchID) (RemoteWallet, error) {
	row := q.queryRow(ctx, q.getRemoteWalletStmt, getRemoteWallet, batchID)
	var i RemoteWallet
	err := row.Scan(
		&i.PeerID,
		&i.AuthToken,
		&i.WalletAddr,
		pq.Array(&i.Multiaddrs),
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.BatchID,
	)
	return i, err
}
