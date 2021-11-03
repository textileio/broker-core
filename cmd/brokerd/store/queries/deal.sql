-- name: CreateDeal :exec
INSERT INTO deals(
    batch_id,
    auction_id,
    bid_id,
    storage_provider_id,
    deal_id,
    deal_expiration,
    error_cause
    ) VALUES(
      $1,
      $2,
      $3,
      $4,
      $5,
      $6,
      $7
      );

-- name: UpdateDeals :execrows
UPDATE deals
SET deal_id = $3,
    deal_expiration = $4,
    error_cause = $5,
    updated_at=CURRENT_TIMESTAMP
WHERE batch_id = $1 AND storage_provider_id = $2;


-- name: GetDeals :many
SELECT * FROM deals WHERE batch_id = $1;

-- name: GetExcludedStorageProviders :many
SELECT DISTINCT d.storage_provider_id
FROM deals d
INNER JOIN batches b ON b.id=d.batch_id
WHERE b.piece_cid=$1 AND
      b.origin=$2 AND
      (b.providers = '{}' OR (b.providers!='{}' AND d.deal_id>0 AND d.deal_expiration>0));
