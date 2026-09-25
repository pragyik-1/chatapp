package routes

import (
	internalMiddleware "chat_app/internal/auth"
	"chat_app/internal/db"
	"chat_app/internal/utils"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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
		AllowedOrigins:   allowedOrigins(),
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

	r.With(rateLimit(10, time.Minute)).Post("/register", registerUser(queries))
	r.With(rateLimit(10, time.Minute)).Post("/login", loginUser(queries, secret))
	r.With(rateLimit(30, time.Minute)).Post("/token/refresh", refreshToken(queries, secret))

	r.Group(func(r chi.Router) {
		r.Use(internalMiddleware.JWTAuth(secret))

		r.Route("/users", func(r chi.Router) {
			r.Get("/search", searchUsers(queries))
			r.Route("/me", func(r chi.Router) {
				r.Get("/", getCurrentUser(queries))
				r.Get("/rooms", getUserRooms(queries))
				r.Get("/settings", getUserSettings(queries))
				r.Patch("/settings", updateUserSettings(queries))
				r.Post("/logout", logoutUser(queries))
				r.Post("/logout-all", logoutUserAll(queries))
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

// allowedOrigins reads the CORS_ORIGINS env var (comma-separated) and falls
// back to the default Vite dev origins when it is unset.
func allowedOrigins() []string {
	if raw := os.Getenv("CORS_ORIGINS"); raw != "" {
		var origins []string
		for _, o := range strings.Split(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
		if len(origins) > 0 {
			return origins
		}
	}
	return []string{"http://localhost:5173", "http://localhost:5174"}
}

// rateLimit is a simple in-memory per-IP token-bucket-ish limiter. It is
// sufficient to blunt brute-force attempts; a shared store would be needed
// for multi-instance deployments.
func rateLimit(requests int, per time.Duration) func(http.Handler) http.Handler {
	type entry struct {
		count int
		reset time.Time
	}
	var mu sync.Mutex
	buckets := make(map[string]*entry)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
				ip = host
			}

			mu.Lock()
			now := time.Now()
			e, ok := buckets[ip]
			if !ok || now.After(e.reset) {
				e = &entry{reset: now.Add(per)}
				buckets[ip] = e
			}
			e.count++
			over := e.count > requests
			mu.Unlock()

			if over {
				utils.WriteError(w, http.StatusTooManyRequests, "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
