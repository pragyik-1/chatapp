package routes

import (
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func searchUsers(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			utils.WriteError(w, http.StatusBadRequest, "query parameter q is required")
			return
		}

		userID, ok := utils.GetUserIDFromContext(r.Context())
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		results, err := queries.SearchUsers(r.Context(), db.SearchUsersParams{
			SearchQuery:   pgtype.Text{String: query, Valid: true},
			ExcludeUserID: pgtype.UUID{Bytes: userID, Valid: true},
			ResultLimit:   20,
		})
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to search users")
			return
		}

		utils.WriteJSON(w, http.StatusOK, results)
	}
}

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
