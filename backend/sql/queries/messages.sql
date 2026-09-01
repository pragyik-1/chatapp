-- name: SendMessage :one
INSERT INTO messages (room_id, sender_id, content, reply_to_id)
VALUES ($1, $2, $3, $4)
RETURNING id, room_id, sender_id, content, reply_to_id, is_edited, edited_at, created_at;

-- name: GetRoomMessages :many
SELECT m.id, m.room_id, m.sender_id, u.username AS sender_name, m.content, m.reply_to_id, m.is_edited, m.edited_at, m.created_at
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE m.room_id = $1
ORDER BY m.created_at ASC
LIMIT $2 OFFSET $3;

-- name: GetMessageByID :one
SELECT id, room_id, sender_id, content, reply_to_id, is_edited, edited_at, created_at
FROM messages
WHERE id = $1;

-- name: EditMessage :one
UPDATE messages
SET content = $1, is_edited = TRUE, edited_at = NOW()
WHERE id = $2 AND sender_id = $3
RETURNING id, room_id, sender_id, content, reply_to_id, is_edited, edited_at, created_at;

-- name: DeleteMessage :exec
DELETE FROM messages
WHERE id = $1 AND sender_id = $2;
