-- name: GetBid :one
SELECT * FROM bids WHERE id = $1;
