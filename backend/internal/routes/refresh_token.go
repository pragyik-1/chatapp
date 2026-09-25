package routes

import (
	"chat_app/internal/auth"
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"encoding/json"
	"net/http"
	"time"
)

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func refreshTokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie("refresh_token"); err == nil && c.Value != "" {
		return c.Value
	}
	var req RefreshRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			return req.RefreshToken
		}
	}
	return ""
}

func refreshToken(queries *db.Queries, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := refreshTokenFromRequest(r)
		if token == "" {
			utils.WriteError(w, http.StatusBadRequest, "refresh token is required")
			return
		}

		stored, err := queries.GetRefreshTokenHashByHash(r.Context(), utils.HashString(token))
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}

		if time.Now().After(stored.ExpiresAt.Time) {
			utils.WriteError(w, http.StatusUnauthorized, "refresh token has expired")
			return
		}

		if stored.IsRevoked.Bool {
			utils.WriteError(w, http.StatusUnauthorized, "refresh token has expired")
			return
		}

		token, err = auth.GenerateJWT(stored.UserID, secret)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to generate token")
			return
		}

		utils.WriteJSON(w, http.StatusOK, map[string]string{"access_token": token})
	}
}
