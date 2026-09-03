package middleware

import (
	"chat_app/internal/utils"
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const UserIDContextKey = "user_id"

func JWTAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.WriteError(w, http.StatusUnauthorized, "missing token")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 {
				utils.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			if parts[0] != "Bearer" {
				utils.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			authHeader = parts[1]
			userID, err := utils.ValidateJWT(authHeader, secret)
			if err != nil {
				utils.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), UserIDContextKey, userID))
			next.ServeHTTP(w, r)
		})
	}
}

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(uuid.UUID)
	return userID, ok
}
