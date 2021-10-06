-- name: CreateOrUpdateStorageProvider :exec
INSERT INTO storage_providers (
    id,
    bidbot_version,
    deal_start_window,
    cid_gravity_configured,
    cid_gravity_strict
    ) VALUES (
      $1,
      $2,
      $3,
      $4,
      $5
      )
  ON CONFLICT (id) DO UPDATE SET
  bidbot_version = $2,
  deal_start_window = $3,
  cid_gravity_configured = $4,
  cid_gravity_strict = $5,
  last_seen_at = CURRENT_TIMESTAMP;

-- name: SetStorageProviderUnhealthy :exec
UPDATE storage_providers SET last_unhealthy_error = $2, last_unhealthy_at = CURRENT_TIMESTAMP WHERE id = $1;
