-- name: CreateOrUpdateAuction :exec
INSERT INTO auctions (
    id,
    batch_id,
    deal_verified,
    excluded_miners,
    status,
    started_at,
    updated_at,
    duration,
    error_cause
    ) VALUES (
      $1,
      $2,
      $3,
      $4,
      $5,
      $6,
      $7,
      $8,
      $9)
  ON CONFLICT (id) DO UPDATE SET
  batch_id = $2,
  deal_verified = $3,
  excluded_miners = $4,
  status = $5,
  started_at = $6,
  updated_at = $7,
  duration = $8,
  error_cause = $9;


-- name: CloseAuction :exec
UPDATE auctions set status = $2, closed_at = $3
WHERE id = $1;

-- name: GetAuction :one
SELECT * FROM auctions WHERE id = $1;
