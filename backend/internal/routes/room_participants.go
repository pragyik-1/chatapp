package routes

import (
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type RoomParticipantRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

func addParticipant(queries db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomIDStr := chi.URLParam(r, "roomId")
		roomID, err := uuid.Parse(roomIDStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid room_id")
			return
		}

		if !_isRoomCreator(r, queries, roomID, w) {
			return
		}

		var req RoomParticipantRequest
		if err := utils.ValidateRequestBody(w, r, &req); err != nil {
			return
		}

		if req.UserID == (uuid.UUID{}) {
			utils.WriteError(w, http.StatusBadRequest, "user_id is required")
			return
		}

		err = queries.AddRoomParticipant(r.Context(), db.AddRoomParticipantParams{
			RoomID: pgtype.UUID{Bytes: roomID, Valid: true},
			UserID: pgtype.UUID{Bytes: req.UserID, Valid: true},
		})
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to add participant")
			return
		}

		utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "participant added successfully"})
	}
}

func removeParticipant(queries db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomIDStr := chi.URLParam(r, "roomId")
		roomID, err := uuid.Parse(roomIDStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid room_id")
			return
		}

		if !_isRoomCreator(r, queries, roomID, w) {
			return
		}

		var req RoomParticipantRequest
		if err := utils.ValidateRequestBody(w, r, &req); err != nil {
			return
		}

		if req.UserID == (uuid.UUID{}) {
			utils.WriteError(w, http.StatusBadRequest, "user_id is required")
			return
		}

		err = queries.RemoveRoomParticipant(r.Context(), db.RemoveRoomParticipantParams{
			RoomID: pgtype.UUID{Bytes: roomID, Valid: true},
			UserID: pgtype.UUID{Bytes: req.UserID, Valid: true},
		})
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to remove participant")
			return
		}

		utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "participant removed successfully"})
	}
}

func getRoomParticipants(queries db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomIDStr := chi.URLParam(r, "roomId")
		roomID, err := uuid.Parse(roomIDStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid room_id")
			return
		}

		userID, ok := utils.GetUserIDFromContext(r.Context())
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if !_isValidParticipant(r, queries, roomID, userID) {
			utils.WriteError(w, http.StatusForbidden, "user is not a participant of this room")
			return
		}

		participants, err := queries.GetRoomParticipants(r.Context(), pgtype.UUID{Bytes: roomID, Valid: true})
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to get room participants")
			return
		}

		utils.WriteJSON(w, http.StatusOK, participants)
	}
}

// _isRoomCreator verifies the authenticated user created the room. It writes
// the error response itself and returns false when the check fails.
func _isRoomCreator(r *http.Request, queries db.Querier, roomID uuid.UUID, w http.ResponseWriter) bool {
	userID, ok := utils.GetUserIDFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}

	room, err := queries.GetRoomByID(r.Context(), pgtype.UUID{Bytes: roomID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "room not found")
			return false
		}
		utils.WriteError(w, http.StatusInternalServerError, "failed to get room")
		return false
	}

	if !room.CreatedBy.Valid || room.CreatedBy.Bytes != userID {
		utils.WriteError(w, http.StatusForbidden, "only the room creator can modify participants")
		return false
	}

	return true
}
