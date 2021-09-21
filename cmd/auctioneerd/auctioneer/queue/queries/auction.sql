-- name: CreateAuction :exec
INSERT INTO auctions(
    id,
    batch_id,
    deal_size,
    deal_duration,
    deal_replication,
    deal_verified,
    fil_epoch_deadline,
    excluded_storage_providers,
    payload_cid,
    car_url,
    car_ipfs_cid,
    car_ipfs_addrs,
    status,
    duration,
    client_address
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
      $15);

-- name: GetAuction :one
SELECT * FROM auctions
WHERE id = $1;

-- name: GetNextReadyToExecute :one
SELECT * FROM auctions
WHERE status = 'queued' OR
(status = 'started' AND extract(epoch from current_timestamp - updated_at) > @stuck_seconds::bigint)
ORDER BY fil_epoch_deadline ASC
FOR UPDATE SKIP LOCKED
LIMIT 1;

-- name: UpdateAuctionStatusAndError :exec
UPDATE auctions
SET status = $2, error_cause = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;
