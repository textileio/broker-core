-- name: CreateBid :exec
INSERT INTO bids (
    auction_id,
    storage_provider_id,
    wallet_addr_sig,
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
      $9);

-- name: WonBid :exec
UPDATE bids SET won_at = $3
WHERE auction_id = $1 and bidder_id = $2;

-- name: AcknowledgedBid :exec
UPDATE bids SET acknowledged_at = $3
WHERE auction_id = $1 and bidder_id = $2;

-- name: ProposalDelivered :exec
UPDATE bids SET proposal_cid = $3, proposal_cid_delivered_at = $4
WHERE auction_id = $1 and bidder_id = $2;

-- name: GetBid :one
SELECT * FROM bids WHERE auction_id = $1 and bidder_id = $2;
