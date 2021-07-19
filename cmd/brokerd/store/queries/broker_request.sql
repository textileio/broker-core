-- name: CreateBrokerRequest :exec
INSERT INTO broker_requests(
    id,
    data_cid,
    status
    ) VALUES ($1, $2, $3);

-- name: GetBrokerRequest :one
SELECT * FROM broker_requests
WHERE id = $1;


-- name: GetBrokerRequestIDs :many
SELECT id FROM broker_requests
WHERE storage_deal_id = $1;

-- name: GetBrokerRequests :many
SELECT * FROM broker_requests
WHERE storage_deal_id = $1;

-- name: BatchUpdateBrokerRequests :many
UPDATE broker_requests
SET status = @status,
    storage_deal_id = @storage_deal_id,
    updated_at = CURRENT_TIMESTAMP
WHERE id = any (@ids::TEXT[])
RETURNING id;

-- name: UpdateBrokerRequestsStatus :exec
UPDATE broker_requests
SET status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE storage_deal_id = $1;

-- name: RebatchBrokerRequests :exec
UPDATE broker_requests
SET rebatch_count = rebatch_count + 1,
    error_cause = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE storage_deal_id = $1;
