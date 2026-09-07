package routes

import (
	"chat_app/internal/auth"
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"net/http"
	"time"
)

type RefreshRequest struct {
	Token string `json:"token"`
}

func refreshToken(queries *db.Queries, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RefreshRequest
		if err := utils.ValidateRequestBody(w, r, &req); err != nil {
			return
		}

		if req.Token == "" {
			utils.WriteError(w, http.StatusBadRequest, "refresh token is required")
			return
		}

		refreshToken, err := queries.GetRefreshTokenByToken(r.Context(), req.Token)
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}

		if time.Now().After(refreshToken.ExpiresAt.Time) {
			utils.WriteError(w, http.StatusUnauthorized, "refresh token has expired")
			return
		}

		userID := refreshToken.UserID

		token, err := auth.GenerateJWT(userID, secret)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to generate token")
			return
		}

		utils.WriteJSON(w, http.StatusOK, map[string]string{"token": token})
	}
}
