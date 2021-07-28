-- name: CreateBatch :exec
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
      $14);

-- name: GetBatch :one
SELECT * FROM batches
WHERE id = $1;

-- name: UpdateBatch :exec
UPDATE batches
SET status = $2,
    piece_cid = $3,
    piece_size = $4,
    updated_at = CURRENT_TIMESTAMP
    WHERE id = $1;

-- name: UpdateBatchStatus :exec
UPDATE batches
SET status = $2,
    updated_at = CURRENT_TIMESTAMP
    WHERE id = $1;

-- name: UpdateBatchStatusAndError :exec
UPDATE batches
SET status = $2,
    error = $3,
    updated_at = CURRENT_TIMESTAMP
    WHERE id = $1;

-- name: CreateBatchTag :exec
INSERT INTO batch_tags (batch_id,key,value) VALUES ($1,$2,$3);
