// Code generated by sqlc. DO NOT EDIT.
// source: bid.sql

package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/textileio/bidbot/lib/auction"
)

const createBid = `-- name: CreateBid :exec
INSERT INTO bids(
    id,
    auction_id,
    wallet_addr_sig,
    storage_provider_id,
    bidder_id,
    ask_price,
    verified_ask_price,
    start_epoch,
    fast_retrieval,
    received_at
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
      $10)
`

type CreateBidParams struct {
	ID                auction.BidID `json:"id"`
	AuctionID         auction.ID    `json:"auctionID"`
	WalletAddrSig     []byte        `json:"walletAddrSig"`
	StorageProviderID string        `json:"storageProviderID"`
	BidderID          string        `json:"bidderID"`
	AskPrice          int64         `json:"askPrice"`
	VerifiedAskPrice  int64         `json:"verifiedAskPrice"`
	StartEpoch        int64         `json:"startEpoch"`
	FastRetrieval     bool          `json:"fastRetrieval"`
	ReceivedAt        time.Time     `json:"receivedAt"`
}

func (q *Queries) CreateBid(ctx context.Context, arg CreateBidParams) error {
	_, err := q.exec(ctx, q.createBidStmt, createBid,
		arg.ID,
		arg.AuctionID,
		arg.WalletAddrSig,
		arg.StorageProviderID,
		arg.BidderID,
		arg.AskPrice,
		arg.VerifiedAskPrice,
		arg.StartEpoch,
		arg.FastRetrieval,
		arg.ReceivedAt,
	)
	return err
}

const getAuctionBids = `-- name: GetAuctionBids :many
SELECT id, auction_id, wallet_addr_sig, storage_provider_id, bidder_id, ask_price, verified_ask_price, start_epoch, fast_retrieval, received_at, won_at, proposal_cid, proposal_cid_delivered_at, proposal_cid_delivery_error, deal_confirmed_at, won_reason, deal_failed_at FROM bids
WHERE auction_id = $1
`

func (q *Queries) GetAuctionBids(ctx context.Context, auctionID auction.ID) ([]Bid, error) {
	rows, err := q.query(ctx, q.getAuctionBidsStmt, getAuctionBids, auctionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Bid
	for rows.Next() {
		var i Bid
		if err := rows.Scan(
			&i.ID,
			&i.AuctionID,
			&i.WalletAddrSig,
			&i.StorageProviderID,
			&i.BidderID,
			&i.AskPrice,
			&i.VerifiedAskPrice,
			&i.StartEpoch,
			&i.FastRetrieval,
			&i.ReceivedAt,
			&i.WonAt,
			&i.ProposalCid,
			&i.ProposalCidDeliveredAt,
			&i.ProposalCidDeliveryError,
			&i.DealConfirmedAt,
			&i.WonReason,
			&i.DealFailedAt,
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

const getAuctionWinningBids = `-- name: GetAuctionWinningBids :many
SELECT id, auction_id, wallet_addr_sig, storage_provider_id, bidder_id, ask_price, verified_ask_price, start_epoch, fast_retrieval, received_at, won_at, proposal_cid, proposal_cid_delivered_at, proposal_cid_delivery_error, deal_confirmed_at, won_reason, deal_failed_at FROM bids
WHERE auction_id = $1 and won_at IS NOT NULL
`

func (q *Queries) GetAuctionWinningBids(ctx context.Context, auctionID auction.ID) ([]Bid, error) {
	rows, err := q.query(ctx, q.getAuctionWinningBidsStmt, getAuctionWinningBids, auctionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Bid
	for rows.Next() {
		var i Bid
		if err := rows.Scan(
			&i.ID,
			&i.AuctionID,
			&i.WalletAddrSig,
			&i.StorageProviderID,
			&i.BidderID,
			&i.AskPrice,
			&i.VerifiedAskPrice,
			&i.StartEpoch,
			&i.FastRetrieval,
			&i.ReceivedAt,
			&i.WonAt,
			&i.ProposalCid,
			&i.ProposalCidDeliveredAt,
			&i.ProposalCidDeliveryError,
			&i.DealConfirmedAt,
			&i.WonReason,
			&i.DealFailedAt,
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

const getRecentWeekFailureRate = `-- name: GetRecentWeekFailureRate :many
WITH b AS (SELECT storage_provider_id,
      extract(epoch from interval '1 weeks') / extract(epoch from current_timestamp - received_at) AS freshness,
      CASE WHEN deal_confirmed_at IS NULL THEN 1 ELSE 0 END failed
    FROM bids
    WHERE received_at > current_timestamp - interval '1 weeks' AND won_at IS NOT NULL
    AND (deal_failed_at IS NOT NULL OR deal_confirmed_at IS NOT NULL))
SELECT b.storage_provider_id, (SUM(b.freshness*b.failed)*1000000/COUNT(*))::bigint AS failure_rate_ppm
FROM b
GROUP BY storage_provider_id
`

type GetRecentWeekFailureRateRow struct {
	StorageProviderID string `json:"storageProviderID"`
	FailureRatePpm    int64  `json:"failureRatePpm"`
}

// here's the logic:
// 1. take all winning bids happened in the recent week until 1 day ago (to exclude the deals waiting to be on-chain), or
// if their deals failed.
// 2. calculate the freshness of each bid, ranging from 1 (1 week ago) to 7 (1 day ago).
// 3. sum up the failed bids (not be able to make a deal), weighed by their freshness, then divide by the number of wins.
func (q *Queries) GetRecentWeekFailureRate(ctx context.Context) ([]GetRecentWeekFailureRateRow, error) {
	rows, err := q.query(ctx, q.getRecentWeekFailureRateStmt, getRecentWeekFailureRate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetRecentWeekFailureRateRow
	for rows.Next() {
		var i GetRecentWeekFailureRateRow
		if err := rows.Scan(&i.StorageProviderID, &i.FailureRatePpm); err != nil {
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

const getRecentWeekMaxOnChainSeconds = `-- name: GetRecentWeekMaxOnChainSeconds :many
SELECT storage_provider_id,
      MAX(extract(epoch from deal_confirmed_at - received_at))::bigint AS max_on_chain_seconds
    FROM bids
    WHERE received_at > current_timestamp - interval '1 weeks' AND deal_confirmed_at IS NOT NULL
GROUP BY storage_provider_id
`

type GetRecentWeekMaxOnChainSecondsRow struct {
	StorageProviderID string `json:"storageProviderID"`
	MaxOnChainSeconds int64  `json:"maxOnChainSeconds"`
}

// get the maximum time in the last week a storage provider making a deal on chain.
func (q *Queries) GetRecentWeekMaxOnChainSeconds(ctx context.Context) ([]GetRecentWeekMaxOnChainSecondsRow, error) {
	rows, err := q.query(ctx, q.getRecentWeekMaxOnChainSecondsStmt, getRecentWeekMaxOnChainSeconds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetRecentWeekMaxOnChainSecondsRow
	for rows.Next() {
		var i GetRecentWeekMaxOnChainSecondsRow
		if err := rows.Scan(&i.StorageProviderID, &i.MaxOnChainSeconds); err != nil {
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

const getRecentWeekWinningRate = `-- name: GetRecentWeekWinningRate :many
WITH b AS (SELECT storage_provider_id,
      extract(epoch from interval '1 weeks') / extract(epoch from current_timestamp - received_at) AS freshness,
      CASE WHEN won_at IS NULL THEN 0 ELSE 1 END winning
    FROM bids
    WHERE received_at < current_timestamp - interval '10 minutes' AND received_at > current_timestamp - interval '1 weeks')
SELECT b.storage_provider_id, (SUM(b.freshness*b.winning)*1000000/COUNT(*))::bigint AS winning_rate_ppm
FROM b
GROUP BY storage_provider_id
`

type GetRecentWeekWinningRateRow struct {
	StorageProviderID string `json:"storageProviderID"`
	WinningRatePpm    int64  `json:"winningRatePpm"`
}

// here's the logic:
// 1. take all received bids happened in the recent week until 10 minutes ago (to exclude the ongoing ones).
// 2. calculate the freshness of each bid, ranging from 1 (1 week ago) to 1008 (10 minutes ago).
// 3. sum up the winning bids, weighed by their freshness, then divide by the number of received bids.
func (q *Queries) GetRecentWeekWinningRate(ctx context.Context) ([]GetRecentWeekWinningRateRow, error) {
	rows, err := q.query(ctx, q.getRecentWeekWinningRateStmt, getRecentWeekWinningRate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetRecentWeekWinningRateRow
	for rows.Next() {
		var i GetRecentWeekWinningRateRow
		if err := rows.Scan(&i.StorageProviderID, &i.WinningRatePpm); err != nil {
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

const updateDealConfirmedAt = `-- name: UpdateDealConfirmedAt :exec
UPDATE bids
SET deal_confirmed_at = CURRENT_TIMESTAMP
WHERE id = $1 AND auction_id = $2
`

type UpdateDealConfirmedAtParams struct {
	ID        auction.BidID `json:"id"`
	AuctionID auction.ID    `json:"auctionID"`
}

func (q *Queries) UpdateDealConfirmedAt(ctx context.Context, arg UpdateDealConfirmedAtParams) error {
	_, err := q.exec(ctx, q.updateDealConfirmedAtStmt, updateDealConfirmedAt, arg.ID, arg.AuctionID)
	return err
}

const updateDealFailedAt = `-- name: UpdateDealFailedAt :exec
UPDATE bids
SET deal_failed_at = CURRENT_TIMESTAMP
WHERE id = $1 AND auction_id = $2
`

type UpdateDealFailedAtParams struct {
	ID        auction.BidID `json:"id"`
	AuctionID auction.ID    `json:"auctionID"`
}

func (q *Queries) UpdateDealFailedAt(ctx context.Context, arg UpdateDealFailedAtParams) error {
	_, err := q.exec(ctx, q.updateDealFailedAtStmt, updateDealFailedAt, arg.ID, arg.AuctionID)
	return err
}

const updateProposalCid = `-- name: UpdateProposalCid :exec
UPDATE bids
SET proposal_cid = $3, proposal_cid_delivered_at = CURRENT_TIMESTAMP
WHERE id = $1 AND auction_id = $2
`

type UpdateProposalCidParams struct {
	ID          auction.BidID  `json:"id"`
	AuctionID   auction.ID     `json:"auctionID"`
	ProposalCid sql.NullString `json:"proposalCid"`
}

func (q *Queries) UpdateProposalCid(ctx context.Context, arg UpdateProposalCidParams) error {
	_, err := q.exec(ctx, q.updateProposalCidStmt, updateProposalCid, arg.ID, arg.AuctionID, arg.ProposalCid)
	return err
}

const updateProposalCidDeliveryError = `-- name: UpdateProposalCidDeliveryError :exec
UPDATE bids
SET proposal_cid_delivery_error = $3
WHERE id = $1 AND auction_id = $2
`

type UpdateProposalCidDeliveryErrorParams struct {
	ID                       auction.BidID  `json:"id"`
	AuctionID                auction.ID     `json:"auctionID"`
	ProposalCidDeliveryError sql.NullString `json:"proposalCidDeliveryError"`
}

func (q *Queries) UpdateProposalCidDeliveryError(ctx context.Context, arg UpdateProposalCidDeliveryErrorParams) error {
	_, err := q.exec(ctx, q.updateProposalCidDeliveryErrorStmt, updateProposalCidDeliveryError, arg.ID, arg.AuctionID, arg.ProposalCidDeliveryError)
	return err
}

const updateWinningBid = `-- name: UpdateWinningBid :execrows
UPDATE bids SET won_reason = $3, won_at = CURRENT_TIMESTAMP
WHERE id = $1 AND auction_id = $2
`

type UpdateWinningBidParams struct {
	ID        auction.BidID  `json:"id"`
	AuctionID auction.ID     `json:"auctionID"`
	WonReason sql.NullString `json:"wonReason"`
}

func (q *Queries) UpdateWinningBid(ctx context.Context, arg UpdateWinningBidParams) (int64, error) {
	result, err := q.exec(ctx, q.updateWinningBidStmt, updateWinningBid, arg.ID, arg.AuctionID, arg.WonReason)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
