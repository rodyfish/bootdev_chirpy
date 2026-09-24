-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, expires_at, revoked_at, user_id)
VALUES (
    $1,
    Now(),
    Now(),
    $2,
    $3,
    $4
)
RETURNING *;

-- name: DeleteAllRefreshTokens :exec
DELETE FROM refresh_tokens;

-- name: GetRefreshToken :one
SELECT *
FROM refresh_tokens
WHERE token = $1;

-- name: GetActiveRefreshToken :one
SELECT *
FROM refresh_tokens
WHERE token = $1
AND revoked_at IS NULL
AND expired_at < Now();

-- name: GetUserFromActiveRefreshToken :one
SELECT u.*
FROM users u JOIN refresh_tokens rt ON rt.user_id = u.id
WHERE rt.token = $1
AND rt.revoked_at IS NULL
AND rt.expires_at >= Now();

-- name: RevokeRefreshToken :one
UPDATE refresh_tokens
SET 
    revoked_at = Now(),
    updated_at = Now()
WHERE token = $1
RETURNING *;