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