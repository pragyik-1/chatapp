package routes

import (
	"chat_app/internal/constants"
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"errors"
	"math/rand/v2"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Color    string `json:"color"`
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

		if req.Color != "" && !slices.Contains(constants.USER_COLORS, req.Color) {
			utils.WriteError(w, http.StatusBadRequest, "invalid color")
			return
		}
		if req.Color == "" {
			req.Color = constants.USER_COLORS[rand.IntN(len(constants.USER_COLORS))]
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

		_, err = queries.CreateUserSettings(r.Context(), db.CreateUserSettingsParams{
			UserID: user.ID,
			Color:  req.Color,
		})
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to register user")
			return
		}

		utils.WriteJSON(w, http.StatusCreated, user)
	}
}
