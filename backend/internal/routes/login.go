package routes

import (
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"net/http"

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

		token, err := utils.GenerateJWT(user.ID, secret)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to generate token")
			return
		}

		utils.WriteJSON(w, http.StatusOK, map[string]string{"token": token})
	}
}
