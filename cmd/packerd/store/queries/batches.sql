-- name: FindOpenBatchWithSpace :one
SELECT *
FROM batches
WHERE status='open' AND
      total_size<=$1 AND
      origin=$2
ORDER BY created_at
FOR UPDATE
LIMIT 1;

-- name: CreateOpenBatch :exec
INSERT INTO batches (batch_id,origin) values ($1,$2);

-- name: AddStorageRequestInBatch :exec
INSERT INTO storage_requests (operation_id, storage_request_id, data_cid, batch_id, size)
VALUES ($1,$2,$3,$4,$5);

-- name: UpdateBatchSize :exec
UPDATE batches
SET total_size=$2, updated_at=CURRENT_TIMESTAMP
WHERE batch_id=$1;

-- name: MoveBatchToStatus :execrows
UPDATE batches
SET status=$2, ready_at=$3, updated_at=CURRENT_TIMESTAMP
WHERE batch_id=$1;

-- name: TimeBasedBatchClosing :execrows
UPDATE batches
SET status='ready', ready_at=CURRENT_TIMESTAMP
WHERE total_size >= 65 AND -- Fundamental minimum size for CommP calculation.
      status='open' AND
      created_at < $1 AND
      origin = 'Textile';

-- name: GetNextReadyBatch :one
UPDATE batches
SET status='executing', updated_at=CURRENT_TIMESTAMP
WHERE batch_id = (SELECT b.batch_id FROM batches b
	          WHERE b.status = 'ready' OR
		        (status='executing' and extract(epoch from current_timestamp - b.updated_at) > @stuck_seconds::bigint)
		  ORDER BY b.ready_at asc
		  FOR UPDATE SKIP LOCKED
	          LIMIT 1)
RETURNING batch_id, total_size, origin, ready_at;

-- name: GetStorageRequestsFromBatch :many
SELECT * FROM storage_requests where batch_id=$1;

-- name: OpenBatchStats :many
SELECT b.origin, 
       count(*) as cid_count,
       (COALESCE(sum(sr.size),0))::bigint as bytes,
       count(DISTINCT sr.batch_id) batches_count
FROM storage_requests sr
JOIN batches b ON b.batch_id=sr.batch_id
WHERE b.status='open'
group by b.origin;

-- name: DoneBatchStats :many
SELECT origin,
       COUNT(*) as batches_count,
       (COALESCE(sum(total_size),0))::bigint as bytes
FROM batches
where status='done'
group by origin;
