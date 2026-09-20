package routes

import (
	internalMiddleware "chat_app/internal/auth"
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

func MakeRouter(queries *db.Queries) *chi.Mux {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set")
	}
	r := chi.NewRouter()
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	r.Use(c.Handler)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "OK"})
	})

	r.Post("/register", registerUser(queries))
	r.Post("/login", loginUser(queries, secret))
	r.Post("/token/refresh", refreshToken(queries, secret))

	r.Group(func(r chi.Router) {
		r.Use(internalMiddleware.JWTAuth(secret))

		r.Route("/users", func(r chi.Router) {
			r.Get("/{userId}/rooms", getUserRooms(queries))
			r.Route("/me", func(r chi.Router) {
				r.Get("/", getCurrentUser(queries))
				r.Get("/settings", getUserSettings(queries))
				r.Patch("/settings", updateUserSettings(queries))
			})
		})
		r.Route("/rooms", func(r chi.Router) {
			r.Post("/create", createRoom(queries))
			r.Get("/{roomId}/participants", getRoomParticipants(queries))
			r.Post("/{roomId}/participants", addParticipant(queries))
			r.Delete("/{roomId}/participants", removeParticipant(queries))

			r.Get("/{roomId}/messages", getRoomMessages(queries))
			r.Post("/{roomId}/messages", sendMessage(queries))
			r.Put("/messages/{messageId}", editMessage(queries))
			r.Delete("/messages/{messageId}", deleteMessage(queries))
		})
	})
	return r
}
