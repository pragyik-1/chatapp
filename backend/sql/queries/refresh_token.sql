-- name: GetRefreshTokenByUserID :one
SELECT id, user_id, token, expires_at, created_at
FROM refresh_tokens
WHERE user_id = $1;

-- name: GetRefreshTokenByToken :one
SELECT id, user_id, token, expires_at, created_at
FROM refresh_tokens
WHERE token = $1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token, expires_at, created_at;

--- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE token = $1;

-- name: RevokeRefreshTokensByUserID :exec
UPDATE refresh_tokens
SET revoked = TRUE
WHERE user_id = $1;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at < NOW() OR revoked = TRUE;