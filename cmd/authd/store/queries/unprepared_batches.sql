-- name: GetAuthToken :one
SELECT * FROM auth_tokens
WHERE token = $1;


