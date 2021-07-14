-- name: CreateMinerDeal :exec
INSERT INTO miner_deals(
    storage_deal_id,
    auction_id,
    bid_id,
    miner_addr,
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
      );

-- name: UpdateMinerDeals :many
UPDATE miner_deals
SET deal_id = $3,
    deal_expiration = $4,
    error_cause = $5
WHERE storage_deal_id = $1 AND miner_addr = $2
RETURNING 0; -- for the caller to count number of rows affected


-- name: GetMinerDeals :many
SELECT * FROM miner_deals WHERE storage_deal_id = $1;
