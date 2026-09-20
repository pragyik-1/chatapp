-- name: GetUserSettings :one
SELECT * FROM user_settings
WHERE user_id = $1;

-- name: CreateUserSettings :one
INSERT INTO user_settings (user_id, color)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateUserSettings :one
UPDATE user_settings
SET color = COALESCE(sqlc.narg('color'), color),
    language = COALESCE(sqlc.narg('language'), language),
    notifications_enabled = COALESCE(sqlc.narg('notifications_enabled'), notifications_enabled),
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;