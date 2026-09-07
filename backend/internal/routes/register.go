package routes

import (
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func registerUser(queries *db.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest
		if err := utils.ValidateRequestBody(w, r, &req); err != nil {
			return
		}

		if req.Username == "" || req.Email == "" || req.Password == "" {
			utils.WriteError(w, http.StatusBadRequest, "username, email, and password are required")
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		user, err := queries.RegisterUser(r.Context(), db.RegisterUserParams{
			Username:     req.Username,
			Email:        req.Email,
			PasswordHash: string(hash),
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				utils.WriteError(w, http.StatusConflict, "a user with this email already exists")
				return
			}
			utils.WriteError(w, http.StatusInternalServerError, "failed to register user")
			return
		}

		utils.WriteJSON(w, http.StatusCreated, user)
	}
}
