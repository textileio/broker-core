-- name: CreateStorageRequest :exec
INSERT INTO storage_requests(
    id,
    data_cid,
    status,
    origin
    ) VALUES ($1,$2,$3,$4);

-- name: GetStorageRequest :one
SELECT * FROM storage_requests
WHERE id = $1;


-- name: GetStorageRequestIDs :many
SELECT id FROM storage_requests
WHERE batch_id = $1;

-- name: GetStorageRequests :many
SELECT * FROM storage_requests
WHERE batch_id = $1;

-- name: BatchUpdateStorageRequests :many
UPDATE storage_requests
SET status = @status,
    batch_id = @batch_id,
    updated_at = CURRENT_TIMESTAMP
WHERE id = any (@ids::TEXT[])
RETURNING id;

-- name: UpdateStorageRequestsStatus :exec
UPDATE storage_requests
SET status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE batch_id = $1;

-- name: RebatchStorageRequests :exec
UPDATE storage_requests
SET rebatch_count = rebatch_count + 1,
    error_cause = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE batch_id = $1;
