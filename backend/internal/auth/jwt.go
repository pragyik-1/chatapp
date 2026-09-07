package auth

import (
	"chat_app/internal/constants"
	"chat_app/internal/utils"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func JWTAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.WriteError(w, http.StatusUnauthorized, "missing token")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				utils.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			tokenString := parts[1]
			userID, err := validateJWT(tokenString, secret)
			if err != nil {
				utils.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), constants.UserIDContextKey, userID))
			next.ServeHTTP(w, r)
		})
	}
}

func validateJWT(tokenString string, secret string) (pgtype.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return pgtype.UUID{}, errors.New("token has expired")
		}
		return pgtype.UUID{}, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UserID, nil
	}

	return pgtype.UUID{}, errors.New("invalid token")
}

type Claims struct {
	UserID pgtype.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID pgtype.UUID, secret string) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func GenerateRefreshToken(userID pgtype.UUID, secret string) (string, error) {
	bytes := make([]byte, 64)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func GenerateTokenPair(userID pgtype.UUID, secret string) (string, string, error) {
	jwtToken, err := GenerateJWT(userID, secret)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := GenerateRefreshToken(userID, secret)
	if err != nil {
		return "", "", err
	}

	return jwtToken, refreshToken, nil
}
