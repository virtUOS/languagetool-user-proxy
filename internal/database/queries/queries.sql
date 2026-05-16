-- name: CreateUser :one
INSERT INTO users (oidc_sub, email, name)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetUserByOIDCSub :one
SELECT * FROM users
WHERE oidc_sub = ?
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = ?
LIMIT 1;

-- name: UpdateUser :exec
UPDATE users
SET email = ?, name = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: CreateAPIKey :one
INSERT INTO api_keys (user_id, key_hash, key_prefix)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys
WHERE key_hash = ?
LIMIT 1;

-- name: GetAPIKeyByUserID :one
SELECT * FROM api_keys
WHERE user_id = ?
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys
WHERE id = ?;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token, expires_at)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetSessionByToken :one
SELECT * FROM sessions
WHERE token = ?
LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < CURRENT_TIMESTAMP;

-- name: ListSessionsByUserID :many
SELECT * FROM sessions
WHERE user_id = ?
AND expires_at > CURRENT_TIMESTAMP;
