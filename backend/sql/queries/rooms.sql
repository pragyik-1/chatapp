-- name: CreateRoom :one
INSERT INTO rooms (name, is_group, created_by)
VALUES ($1, $2, $3)
RETURNING id, name, is_group, created_by, created_at;

-- name: GetRoomByID :one
SELECT id, name, is_group, created_by, created_at
FROM rooms
WHERE id = $1;

-- name: AddRoomParticipant :exec
INSERT INTO room_participants (room_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveRoomParticipant :exec
DELETE FROM room_participants
WHERE room_id = $1 AND user_id = $2;

-- name: GetRoomParticipants :many
SELECT u.id, u.username, u.email, u.status, u.last_seen, us.color, rp.joined_at
FROM room_participants rp
JOIN users u ON rp.user_id = u.id
LEFT JOIN user_settings us ON us.user_id = u.id
WHERE rp.room_id = $1;

-- name: IsParticipant :one
SELECT EXISTS (
    SELECT 1 FROM room_participants
    WHERE room_id = $1 AND user_id = $2
);