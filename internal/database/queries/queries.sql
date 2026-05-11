-- name: CreateUser :one
INSERT INTO users (oidc_sub, email, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByOIDCSub :one
SELECT * FROM users
WHERE oidc_sub = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: UpdateUser :exec
UPDATE users
SET email = $2, name = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: CreateAPIKey :one
INSERT INTO api_keys (user_id, key_hash, key_prefix)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys
WHERE key_hash = $1
LIMIT 1;

-- name: GetAPIKeyByUserID :one
SELECT * FROM api_keys
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys
WHERE id = $1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSessionByToken :one
SELECT * FROM sessions
WHERE token = $1
LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < CURRENT_TIMESTAMP;

-- name: ListSessionsByUserID :many
SELECT * FROM sessions
WHERE user_id = $1
AND expires_at > CURRENT_TIMESTAMP;
