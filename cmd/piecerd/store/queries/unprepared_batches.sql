-- name: CreateUnpreparedBatch :exec
INSERT INTO unprepared_batches(
    batch_id,
    data_cid
 ) VALUES ($1, $2);

-- name: GetNextPending :one
UPDATE unprepared_batches
SET status = 'executing', updated_at = CURRENT_TIMESTAMP
WHERE batch_id = (SELECT ub.batch_id FROM unprepared_batches ub
            WHERE (ub.ready_at < CURRENT_TIMESTAMP AND ub.status = 'pending') OR
	          (ub.status='executing' and extract(epoch from current_timestamp - ub.updated_at) > @stuckSeconds::bigint)
                  ORDER BY ub.ready_at asc 
                  FOR UPDATE SKIP LOCKED
                  LIMIT 1)
RETURNING *;

-- name: MoveToStatus :execrows
UPDATE unprepared_batches 
SET status = $3, updated_at = CURRENT_TIMESTAMP, ready_at=$2
WHERE batch_id = $1;
