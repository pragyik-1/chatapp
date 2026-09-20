-- name: GetRefreshTokenByUserID :one
SELECT id, user_id, token_hash, expires_at, created_at, is_revoked
FROM refresh_tokens
WHERE user_id = $1;

-- name: GetRefreshTokenByToken :one
SELECT id, user_id, token_hash, expires_at, created_at, is_revoked
FROM refresh_tokens
WHERE token_hash = $1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, created_at, is_revoked;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE token_hash = $1;

-- name: RevokeRefreshTokensByUserID :exec
UPDATE refresh_tokens
SET is_revoked = TRUE
WHERE user_id = $1;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at < NOW() OR is_revoked = TRUE;