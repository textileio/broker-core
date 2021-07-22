-- name: FindOpenBatchWithSpace :one
SELECT *
FROM batches
WHERE status = 'open' AND
      total_size<=$1
ORDER BY created_at
FOR UPDATE
LIMIT 1;

-- name: CreateOpenBatch :exec
INSERT INTO batches (batch_id) values ($1);

-- name: AddStorageRequestInBatch :exec
INSERT INTO storage_requests (operation_id, storage_request_id, data_cid, batch_id, size)
VALUES ($1,$2,$3,$4,$5);

-- name: UpdateBatchSize :exec
UPDATE batches
SET total_size=$2
WHERE batch_id=$1;

-- name: MoveBatchToStatus :execrows
UPDATE batches
SET status=$2, updated_at = CURRENT_TIMESTAMP
WHERE batch_id=$1;

-- name: GetNextReadyBatch :one
UPDATE batches
SET status = 'executing', updated_at = CURRENT_TIMESTAMP
WHERE batch_id = (SELECT b.batch_id FROM batches b
	          WHERE b.status = 'ready'
		  ORDER BY b.updated_at asc
		  FOR UPDATE SKIP LOCKED
	          LIMIT 1)
RETURNING batch_id, total_size;

-- name: GetStorageRequestFromBatch :many
SELECT * FROM storage_requests where batch_id=$1;

-- name: OpenBatchStats :one
SELECT count(*) as srs_count,
       sum(sr.size) as batches_bytes,
       count(DISTINCT batch_id) batches_count
FROM storage_requests sr
JOIN batches b ON b.batch_id=sr.batch_id
WHERE b.status='open';

-- name: DoneBatchStats :one
SELECT count(*) as done_batch_count,
       sum(size) as done_batch_bytes
FROM batches
where status='done';
