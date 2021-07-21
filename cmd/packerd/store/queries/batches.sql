-- name: FindOpenBatchWithSpace :one
SELECT *
FROM batches
WHERE status = 'open' AND
      total_size <= $1
ORDER BY created_at
FOR UPDATE
LIMIT 1;

-- name: CreateOpenBatch :exec
INSERT INTO batches (batch_id) values ($1);

-- name: AddStorageRequestInBatch :exec
INSERT INTO storage_requests (operation_id, storage_request_id, data_cid, batch_id)
VALUES ($1, $2, $3, $4);

-- name: UpdateBatchSize :exec
UPDATE batches
SET total_size = $2
WHERE batch_id = $1;

-- name: MoveBatchToStatus :execrows
UPDATE batches
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE batch_id = $1;


