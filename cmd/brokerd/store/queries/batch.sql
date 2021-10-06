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
      $15);

-- name: CreateBatchManifest :exec
INSERT INTO batch_manifests(
   batch_id,
   manifest
   ) VALUES (
     $1,
     $2);

-- name: GetBatchManifest :one
SELECT * FROM batch_manifests
WHERE batch_id=$1;

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

-- name: GetBatchTags :many
SELECT * FROM batch_tags
WHERE batch_id=$1;

-- name: GetBatchRemoteWallet :one
SELECT * FROM batch_remote_wallet
WHERE batch_id=$1;

-- name: CreateBatchRemoteWallet :exec
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
	$5);
