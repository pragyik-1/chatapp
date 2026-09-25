-- name: RegisterUser :one
INSERT INTO users (username, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, username, email, status, last_seen, created_at;

-- name: RegisterUserWithSettings :one
WITH new_user AS (
    INSERT INTO users (username, email, password_hash)
    VALUES ($1, $2, $3)
    RETURNING *
),
settings_insert AS (
    INSERT INTO user_settings (user_id, color)
    SELECT id, $4 FROM new_user
)
SELECT id, username, email, status, last_seen, created_at FROM new_user;

-- name: GetUserByID :one
SELECT id, username, email, status, last_seen, created_at
FROM users
WHERE id = $1;

-- name: GetUserRooms :many
SELECT r.id, r.name, r.is_group, r.created_by, r.created_at
FROM rooms r
JOIN room_participants rp ON r.id = rp.room_id
WHERE rp.user_id = $1
ORDER BY r.created_at DESC;

-- name: GetUserByEmail :one
SELECT id, username, email, status, last_seen, created_at, password_hash
FROM users
WHERE email = $1;

-- name: SearchUsers :many
SELECT u.id, u.username, u.email, u.status, u.last_seen, u.created_at, s.color
FROM users u
LEFT JOIN user_settings s ON s.user_id = u.id
WHERE (
    u.username ILIKE '%' || @search_query || '%'
    OR u.email = @search_query
    OR u.id::text = @search_query
)
AND u.id <> @exclude_user_id
ORDER BY u.username
LIMIT @result_limit;