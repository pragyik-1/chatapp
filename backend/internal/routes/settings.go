package routes

import (
	"chat_app/internal/constants"
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5/pgtype"
)

type updateUserSettingsRequest struct {
	Color                *string `json:"color"`
	Theme                *string `json:"theme"`
	Language             *string `json:"language"`
	NotificationsEnabled *bool   `json:"notifications_enabled"`
}

func getUserSettings(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := utils.GetUserIDFromContext(r.Context())
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		settings, err := queries.GetUserSettings(r.Context(), pgtype.UUID{Bytes: userID, Valid: true})
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to get user settings")
			return
		}

		utils.WriteJSON(w, http.StatusOK, settings)
	}
}

func updateUserSettings(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := utils.GetUserIDFromContext(r.Context())
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req updateUserSettingsRequest
		if err := utils.ValidateRequestBody(w, r, &req); err != nil {
			return
		}

		if req.Color != nil && !slices.Contains(constants.USER_COLORS, *req.Color) {
			utils.WriteError(w, http.StatusBadRequest, "invalid color")
			return
		}
		if req.Theme != nil && !slices.Contains(constants.USER_THEMES, *req.Theme) {
			utils.WriteError(w, http.StatusBadRequest, "invalid theme")
			return
		}

		params := db.UpdateUserSettingsParams{
			UserID: pgtype.UUID{Bytes: userID, Valid: true},
		}
		if req.Color != nil {
			params.Color = pgtype.Text{String: *req.Color, Valid: true}
		}
		if req.Language != nil {
			params.Language = pgtype.Text{String: *req.Language, Valid: true}
		}
		if req.NotificationsEnabled != nil {
			params.NotificationsEnabled = pgtype.Bool{Bool: *req.NotificationsEnabled, Valid: true}
		}

		settings, err := queries.UpdateUserSettings(r.Context(), params)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to update user settings")
			return
		}

		utils.WriteJSON(w, http.StatusOK, settings)
	}
}
