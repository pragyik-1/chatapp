package routes

import (
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type GetUserRoomsRequest struct {
	UserId uuid.UUID `json:"user_id"`
}

func getUserRooms(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDStr := chi.URLParam(r, "userId")
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid user_id")
			return
		}

		rooms, err := queries.GetUserRooms(r.Context(), pgtype.UUID{Bytes: userID, Valid: true})
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to get user rooms")
			return
		}

		utils.WriteJSON(w, http.StatusOK, rooms)
	}
}

func getCurrentUser(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := utils.GetUserIDFromContext(r.Context())
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		user, err := queries.GetUserByID(r.Context(), pgtype.UUID{Bytes: userID, Valid: true})

		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid user_id")
			return
		}

		utils.WriteJSON(w, http.StatusOK, user)
	}
}
