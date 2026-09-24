package routes

import (
	"chat_app/internal/auth"
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

const refreshTokenCookieName = "refresh_token"

func loginUser(queries *db.Queries, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := utils.ValidateRequestBody(w, r, &req); err != nil {
			return
		}

		if req.Email == "" || req.Password == "" {
			utils.WriteError(w, http.StatusBadRequest, "email and password are required")
			return
		}

		user, err := queries.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			if err == pgx.ErrNoRows {
				utils.WriteError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}
			utils.WriteError(w, http.StatusInternalServerError, "failed to get user")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			utils.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		accessToken, refreshToken, err := auth.GenerateTokenPair(user.ID, secret)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to generate token pair")
			return
		}

		refreshTokenHash := utils.HashString(refreshToken)
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
			Name:     refreshTokenCookieName,
			Value:    refreshToken,
			Path:     "/",
			Expires:  expiresAt,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		utils.WriteJSON(w, http.StatusOK, map[string]string{"access_token": accessToken})
	}
}

func logoutUserAll(queries db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := utils.GetUserIDFromContext(r.Context())
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if err := queries.RevokeRefreshTokensByUserID(r.Context(), pgtype.UUID{Bytes: userID, Valid: true}); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to revoke refresh token")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     refreshTokenCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
		})

		w.WriteHeader(http.StatusNoContent)
	}
}

func logoutUser(queries db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LogoutRequest
		if err := utils.ValidateRequestBody(w, r, &req); err != nil {
			return
		}

		if req.RefreshToken == "" {
			utils.WriteError(w, http.StatusBadRequest, "no refresh token provided")
			return
		}

		if err := queries.RevokeRefreshToken(r.Context(), req.RefreshToken); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to revoke refresh token")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     refreshTokenCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
		})

		w.WriteHeader(http.StatusNoContent)
	}
}
