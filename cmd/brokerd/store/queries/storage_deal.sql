-- name: CreateStorageDeal :exec
INSERT INTO storage_deals(
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
    piece_size
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
      $13
      );

-- name: GetStorageDeal :one
SELECT * FROM storage_deals
WHERE id = $1;

-- name: UpdateStorageDeal :exec
UPDATE storage_deals
SET status = $2,
    piece_cid = $3,
    piece_size = $4,
    updated_at = CURRENT_TIMESTAMP
    WHERE id = $1;

-- name: UpdateStorageDealStatus :exec
UPDATE storage_deals
SET status = $2,
    updated_at = CURRENT_TIMESTAMP
    WHERE id = $1;

-- name: UpdateStorageDealStatusAndError :exec
UPDATE storage_deals
SET status = $2,
    error = $3,
    updated_at = CURRENT_TIMESTAMP
    WHERE id = $1;
