-- name: FindOpenBatchWithSpace :one
SELECT *
FROM batches
WHERE status = 'open' AND
      total_size <= $1
ORDER BY created_at;

-- name: CreateOpenBatch :exec
INSERT INTO batches (batch_id) values ($1);

-- name: AddStorageRequestInBatch :exec
INSERT INTO storage_requests (storage_request_id,data_cid)
VALUES ($1, $2);

-- name: MoveBatchToStatus :exec
UPDATE batches
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE batch_id = $1;


