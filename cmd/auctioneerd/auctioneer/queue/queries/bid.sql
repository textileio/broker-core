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
