-- name: CreateBid :exec
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
      $10);

-- name: GetAuctionBids :many
SELECT * FROM bids
WHERE auction_id = $1;

-- name: GetAuctionWinningBids :many
SELECT * FROM bids
WHERE auction_id = $1 and won_at IS NOT NULL;

-- name: GetRecentWeekFailureRate :many
-- here's the logic:
-- 1. take all winning bids happened in the recent week until 1 day ago (to exclude the deals waiting to be on-chain).
-- 2. calculate the freshness of each bid, ranging from 1 (1 week ago) to 7 (1 day ago).
-- 3. sum up the failed bids (not be able to make a deal), weighed by their freshness, then divide by the number of wins.
WITH b AS (SELECT storage_provider_id,
      extract(epoch from interval '1 weeks') / extract(epoch from current_timestamp - received_at) AS freshness,
      CASE WHEN deal_confirmed_at IS NULL THEN 1 ELSE 0 END failed
    FROM bids
    WHERE received_at < current_timestamp - interval '1 days' AND received_at > current_timestamp - interval '1 weeks' AND won_at IS NOT NULL)
SELECT b.storage_provider_id, (SUM(b.freshness*b.failed)*1000000/COUNT(*))::bigint AS failure_rate_ppm
FROM b
GROUP BY storage_provider_id ORDER by failure_rate_ppm;

-- name: GetRecentWeekWinningRate :many
-- here's the logic:
-- 1. take all received bids happened in the recent week until 10 minutes ago (to exclude the ongoing ones).
-- 2. calculate the freshness of each bid, ranging from 1 (1 week ago) to 1008 (10 minutes ago).
-- 3. sum up the winning bids, weighed by their freshness, then divide by the number of received bids.
WITH b AS (SELECT storage_provider_id,
      extract(epoch from interval '1 weeks') / extract(epoch from current_timestamp - received_at) AS freshness,
      CASE WHEN won_at IS NULL THEN 0 ELSE 1 END winning
    FROM bids
    WHERE received_at < current_timestamp - interval '10 minutes' AND received_at > current_timestamp - interval '1 weeks')
SELECT b.storage_provider_id, (SUM(b.freshness*b.winning)*1000000/COUNT(*))::bigint AS winning_rate_ppm
FROM b
GROUP BY storage_provider_id ORDER by winning_rate_ppm;

-- name: UpdateBidsWonAt :many
UPDATE bids SET won_at = CURRENT_TIMESTAMP
WHERE id = ANY(@bid_ids::text[]) AND auction_id = @auction_id
RETURNING id;

-- name: UpdateProposalCid :exec
UPDATE bids
SET proposal_cid = $3, proposal_cid_delivered_at = CURRENT_TIMESTAMP
WHERE id = $1 AND auction_id = $2;

-- name: UpdateProposalCidDeliveryError :exec
UPDATE bids
SET proposal_cid_delivery_error = $3
WHERE id = $1 AND auction_id = $2;

-- name: UpdateDealConfirmedAt :exec
UPDATE bids
SET deal_confirmed_at = CURRENT_TIMESTAMP
WHERE id = $1 AND auction_id = $2;
