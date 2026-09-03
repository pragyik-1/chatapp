package routes

import (
	"chat_app/internal/db"
	internalMiddleware "chat_app/internal/middleware"
	"chat_app/internal/utils"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func MakeRouter(queries *db.Queries) *chi.Mux {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set")
	}
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "OK"})
	})

	r.Post("/register", registerUser(queries))

	r.Group(func(r chi.Router) {
		r.Use(internalMiddleware.JWTAuth(secret))

		r.Route("/user", func(r chi.Router) {
			r.Get("/{userId}/rooms", getUserRooms(queries))
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
