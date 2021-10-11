-- name: GetMarketDealStatusForID :one
SELECT * FROM market_deal_status WHERE id = $1;

-- name: GetMarketDealStatusForType :one
SELECT * FROM market_deal_status WHERE type = @t;
