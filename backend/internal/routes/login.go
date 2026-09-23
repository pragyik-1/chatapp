package routes

import (
	"chat_app/internal/auth"
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func loginUser(queries *db.Queries, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := utils.ParseJSONRequestBody(w, r, &req); err != nil {
			return
		}

		if req.Email == "" || req.Password == "" {
			utils.WriteError(w, http.StatusBadRequest, "email and password are required")
			return
		}

		user, err := queries.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to get user")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			utils.WriteError(w, http.StatusUnauthorized, "invalid password")
			return
		}

		accessToken, refreshToken, err := auth.GenerateTokenPair(user.ID, secret)
		refreshTokenHash := utils.HashString(refreshToken)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to generate token pair")
			return
		}

		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		if _, err := queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
			UserID:    user.ID,
			TokenHash: refreshTokenHash,
			ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		}); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to store refresh token")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token_hash",
			Value:    refreshTokenHash,
			Expires:  expiresAt,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		utils.WriteJSON(w, http.StatusOK, map[string]string{"access_token": accessToken, "refresh_token_hash": refreshTokenHash})
	}
}
