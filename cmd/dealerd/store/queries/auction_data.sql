-- name: CreateAuctionData :exec
INSERT INTO auction_data(
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
      $5
      );

-- name: GetAuctionData :one
SELECT * FROM auction_data
WHERE batch_id = $1;

-- name: RemoveAuctionData :exec
DELETE FROM auction_data WHERE batch_id = $1;

-- name: CreateRemoteWallet :exec
INSERT INTO remote_wallet(
   batch_id,
   peer_id,
   auth_token,
   wallet_addr,
   multiaddrs
   ) VALUES (
   $1,
   $2,
   $3,
   $4,
   $5);

-- name: GetRemoteWallet :one
SELECT * FROM remote_wallet
where batch_id = $1;

-- name: RemoveRemoteWallet :exec
DELETE FROM remote_wallet WHERE batch_id = $1;
