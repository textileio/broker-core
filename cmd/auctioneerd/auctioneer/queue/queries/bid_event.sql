-- name: CreateBidEvent :exec
INSERT INTO bid_events (
    bid_id,
    event_type,
    attempts,
    error,
    happened_at,
    received_at ) VALUES (
      $1,
      $2,
      $3,
      $4,
      $5,
      $6
      );
