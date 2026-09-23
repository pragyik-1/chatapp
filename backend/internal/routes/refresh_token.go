package routes

import (
	"chat_app/internal/auth"
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"net/http"
	"time"
)

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token_hash"`
}

func refreshToken(queries *db.Queries, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RefreshRequest
		if err := utils.ValidateRequestBody(w, r, &req); err != nil {
			return
		}

		if req.RefreshToken == "" {
			utils.WriteError(w, http.StatusBadRequest, "refresh token is required")
			return
		}

		refreshToken, err := queries.GetRefreshTokenByToken(r.Context(), req.RefreshToken)
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}

		if time.Now().After(refreshToken.ExpiresAt.Time) {
			utils.WriteError(w, http.StatusUnauthorized, "refresh token has expired")
			return
		}

		if refreshToken.IsRevoked.Bool {
			utils.WriteError(w, http.StatusUnauthorized, "refresh token has expired")
			return
		}

		userID := refreshToken.UserID

		token, err := auth.GenerateJWT(userID, secret)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to generate token")
			return
		}

		utils.WriteJSON(w, http.StatusOK, map[string]string{"access_token": token})
	}
}

func revokeRefreshToken(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RefreshRequest
		if err := utils.ValidateRequestBody(w, r, &req); err != nil {
			return
		}

		if req.RefreshToken == "" {
			utils.WriteError(w, http.StatusBadRequest, "refresh token is required")
			return
		}

		err := queries.RevokeRefreshToken(r.Context(), req.RefreshToken)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to revoke refresh token")
			return
		}

		utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "refresh token revoked successfully"})
	}
}
