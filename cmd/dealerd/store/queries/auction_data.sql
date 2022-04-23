-- name: CreateAuctionData :exec
INSERT INTO auction_data(
    id,
    batch_id,
    payload_cid,
    piece_cid,
    piece_size,
    duration,
    car_url
    ) VALUES (
      $1,
      $2,
      $3,
      $4,
      $5,
      $6,
      $7
      );

-- name: GetAuctionData :one
SELECT * FROM auction_data
WHERE id = $1;

-- name: IsBoostAllowed :one
SELECT * FROM boost_whitelist
WHERE storage_provider_id=$1;

-- name: CreateRemoteWallet :exec
INSERT INTO remote_wallet(
   auction_data_id,
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
where auction_data_id = $1;
