-- name: CreateAuctionData :exec
INSERT INTO auction_data(
    id,
    batch_id,
    payload_cid,
    piece_cid,
    piece_size,
    duration
    ) VALUES (
      $1,
      $2,
      $3,
      $4,
      $5,
      $6
      );

-- name: GetAuctionData :one
SELECT * FROM auction_data
WHERE id = $1;

-- name: RemoveAuctionData :exec
DELETE FROM auction_data WHERE id = $1;
