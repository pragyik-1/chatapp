package routes

import (
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func getUserRooms(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := utils.GetUserIDFromContext(r.Context())
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
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
			if errors.Is(err, pgx.ErrNoRows) {
				utils.WriteError(w, http.StatusNotFound, "user not found")
				return
			}
			utils.WriteError(w, http.StatusInternalServerError, "failed to get user")
			return
		}

		utils.WriteJSON(w, http.StatusOK, user)
	}
}
