package routes

import (
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type SendMessageRequest struct {
	RoomID   uuid.UUID  `json:"room_id"`
	SenderID uuid.UUID  `json:"sender_id"`
	Content  string     `json:"content"`
	ReplyTo  *uuid.UUID `json:"reply_to_id"`
}

type EditMessageRequest struct {
	SenderID uuid.UUID `json:"sender_id"`
	Content  string    `json:"content"`
}

type DeleteMessageRequest struct {
	SenderID uuid.UUID `json:"sender_id"`
}

func sendMessage(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		if err := utils.ParseJSONRequestBody(w, r, &req); err != nil {
			return
		}

		if req.RoomID == (uuid.UUID{}) || req.SenderID == (uuid.UUID{}) || req.Content == "" {
			utils.WriteError(w, http.StatusBadRequest, "room_id, sender_id, and content are required")
			return
		}

		// Validate participant
		isParticipant, err := queries.IsParticipant(r.Context(), db.IsParticipantParams{
			RoomID: pgtype.UUID{Bytes: req.RoomID, Valid: true},
			UserID: pgtype.UUID{Bytes: req.SenderID, Valid: true},
		})
		if err != nil || !isParticipant {
			utils.WriteError(w, http.StatusForbidden, "user is not a participant of this room")
			return
		}

		var replyToID pgtype.UUID
		if req.ReplyTo != nil {
			reply, err := queries.GetMessageByID(r.Context(), pgtype.UUID{Bytes: *req.ReplyTo, Valid: true})
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					utils.WriteError(w, http.StatusBadRequest, "reply_to_id refers to a non-existent message")
					return
				}
				utils.WriteError(w, http.StatusInternalServerError, "failed to send message")
				return
			}
			if !reply.RoomID.Valid || reply.RoomID.Bytes != req.RoomID {
				utils.WriteError(w, http.StatusBadRequest, "reply_to_id does not belong to this room")
				return
			}
			replyToID = pgtype.UUID{Bytes: *req.ReplyTo, Valid: true}
		}

		msg, err := queries.SendMessage(r.Context(), db.SendMessageParams{
			RoomID:    pgtype.UUID{Bytes: req.RoomID, Valid: true},
			SenderID:  pgtype.UUID{Bytes: req.SenderID, Valid: true},
			Content:   req.Content,
			ReplyToID: replyToID,
		})
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to send message")
			return
		}

		utils.WriteJSON(w, http.StatusCreated, msg)
	}
}

func getRoomMessages(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomIDStr := chi.URLParam(r, "roomId")
		roomID, err := uuid.Parse(roomIDStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid room_id")
			return
		}

		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")

		limit := int32(50)
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = int32(l)
		}

		offset := int32(0)
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = int32(o)
		}

		messages, err := queries.GetRoomMessages(r.Context(), db.GetRoomMessagesParams{
			RoomID: pgtype.UUID{Bytes: roomID, Valid: true},
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to retrieve messages")
			return
		}

		utils.WriteJSON(w, http.StatusOK, messages)
	}
}

func editMessage(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		messageIDStr := chi.URLParam(r, "messageId")
		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid message_id")
			return
		}

		var req EditMessageRequest
		if err := utils.ParseJSONRequestBody(w, r, &req); err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.SenderID == (uuid.UUID{}) || req.Content == "" {
			utils.WriteError(w, http.StatusBadRequest, "sender_id and content are required")
			return
		}

		existing, err := queries.GetMessageByID(r.Context(), pgtype.UUID{Bytes: messageID, Valid: true})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				utils.WriteError(w, http.StatusNotFound, "message not found")
				return
			}
			utils.WriteError(w, http.StatusInternalServerError, "failed to edit message")
			return
		}

		if !existing.SenderID.Valid || existing.SenderID.Bytes != req.SenderID {
			utils.WriteError(w, http.StatusForbidden, "you are not the sender of this message")
			return
		}

		msg, err := queries.EditMessage(r.Context(), db.EditMessageParams{
			Content:  req.Content,
			ID:       pgtype.UUID{Bytes: messageID, Valid: true},
			SenderID: pgtype.UUID{Bytes: req.SenderID, Valid: true},
		})
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to edit message")
			return
		}

		utils.WriteJSON(w, http.StatusOK, msg)
	}
}

func deleteMessage(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		messageIDStr := chi.URLParam(r, "messageId")
		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid message_id")
			return
		}

		var req DeleteMessageRequest
		if err := utils.ParseJSONRequestBody(w, r, &req); err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.SenderID == (uuid.UUID{}) {
			utils.WriteError(w, http.StatusBadRequest, "sender_id is required")
			return
		}

		existing, err := queries.GetMessageByID(r.Context(), pgtype.UUID{Bytes: messageID, Valid: true})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				utils.WriteError(w, http.StatusNotFound, "message not found")
				return
			}
			utils.WriteError(w, http.StatusInternalServerError, "failed to delete message")
			return
		}

		if !existing.SenderID.Valid || existing.SenderID.Bytes != req.SenderID {
			utils.WriteError(w, http.StatusForbidden, "you are not the sender of this message")
			return
		}

		err = queries.DeleteMessage(r.Context(), db.DeleteMessageParams{
			ID:       pgtype.UUID{Bytes: messageID, Valid: true},
			SenderID: pgtype.UUID{Bytes: req.SenderID, Valid: true},
		})
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to delete message")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
