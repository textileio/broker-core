-- name: CreateUnpinJob :exec
INSERT INTO unpin_jobs(id, cid, type) VALUES ($1, $2, $3);

-- name: NextUnpinJob :one
UPDATE unpin_jobs
SET executing = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE id = (SELECT id FROM unpin_jobs
    WHERE unpin_jobs.ready_at < CURRENT_TIMESTAMP AND NOT executing
    ORDER BY unpin_jobs.ready_at
    FOR UPDATE SKIP LOCKED
    asc LIMIT 1)
RETURNING *;

-- name: UnpinJobToPending :exec
UPDATE unpin_jobs
SET executing = FALSE, ready_at = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND executing;

-- name: DeleteExecutingUnpinJob :exec
DELETE FROM unpin_jobs WHERE id = $1 AND executing;
