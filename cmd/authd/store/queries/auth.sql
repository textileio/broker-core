-- name: GetAuthToken :one
SELECT * FROM auth_tokens
WHERE token = $1;

-- name: CreateAuthToken :exec
INSERT INTO auth_tokens (token,identity,origin)
VALUES ($1,$2,$3);
