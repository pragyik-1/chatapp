package routes

import (
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type RoomParticipantRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

func addParticipant(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomIDStr := chi.URLParam(r, "roomId")
		roomID, err := uuid.Parse(roomIDStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid room_id")
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

func removeParticipant(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomIDStr := chi.URLParam(r, "roomId")
		roomID, err := uuid.Parse(roomIDStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid room_id")
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

func getRoomParticipants(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomIDStr := chi.URLParam(r, "roomId")
		roomID, err := uuid.Parse(roomIDStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid room_id")
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
