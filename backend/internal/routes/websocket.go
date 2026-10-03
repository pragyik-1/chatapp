package routes

import (
	"chat_app/internal/auth"
	"chat_app/internal/constants"
	"chat_app/internal/db"
	"chat_app/internal/hub"
	"chat_app/internal/utils"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var errNotParticipant = errors.New("user is not a participant of this room")

// realtimeEvent is the envelope for every server-to-client frame. The type is
// one of the constants.Event* values and the data payload is the REST resource
// the event concerns, so a client can reuse the same types it already has.
type realtimeEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// clientFrame is the only shape a client may send. Subscriptions are delivery
// control, not application writes: messages are still created, edited, and
// deleted through the REST endpoints.
type clientFrame struct {
	Type   string     `json:"type"`
	RoomID *uuid.UUID `json:"room_id"`
}

// The access token arrives as a query parameter because a browser cannot set an
// Authorization header on a WebSocket handshake. It is registered outside the
// JWTAuth group and validated here instead.
func serveWebSocket(queries db.Querier, h *hub.Hub, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(r.Context())

		token := r.URL.Query().Get(constants.WSAuthQueryParam)

		if token == "" {
			utils.WriteError(w, http.StatusUnauthorized, "missing token")
			return
		}

		claims, err := auth.ValidateJWT(token, secret)
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		userID := uuid.UUID(claims.Bytes)

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: allowedOrigins(),
		})
		if err != nil {
			log.Printf("websocket accept failed: %v (req=%s)", err, reqID)
			return
		}

		client, err := h.Serve(r.Context(), conn, userID)
		if err != nil {
			closeStatus := websocket.StatusTryAgainLater
			if !errors.Is(err, hub.ErrShuttingDown) {
				closeStatus = websocket.StatusPolicyViolation
			}
			if closeErr := conn.Close(closeStatus, "server unavailable"); closeErr != nil {
				log.Printf("websocket close after %v failed: %v (req=%s)", err, closeErr, reqID)
			}
			return
		}
		defer h.Remove(client)

		_readLoop(r, queries, h, client, conn, userID, reqID)
	}
}

// _readLoop consumes client frames until the connection ends, translating them
// into hub subscriptions. It writes no response of its own: once the handshake
// succeeds the only way to reject a client is a close frame.
func _readLoop(r *http.Request, queries db.Querier, h *hub.Hub, client *hub.Client, conn *websocket.Conn, userID uuid.UUID, reqID string) {
	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			// A read error is how a normal disconnect surfaces: the client
			// closed, went away, or failed the ping liveness check. It is not
			// a server fault, so it is not logged
			return
		}

		var frame clientFrame
		if err := json.Unmarshal(data, &frame); err != nil {
			_closeWebSocket(conn, websocket.StatusPolicyViolation, "malformed frame", reqID)
			return
		}

		switch frame.Type {
		case constants.WSFrameSubscribe, constants.WSFrameUnsubscribe:
		default:
			_closeWebSocket(conn, websocket.StatusPolicyViolation, "unsupported frame type", reqID)
			return
		}

		if frame.RoomID == nil || *frame.RoomID == (uuid.UUID{}) {
			_closeWebSocket(conn, websocket.StatusPolicyViolation, "room_id is required", reqID)
			return
		}
		roomID := *frame.RoomID

		if frame.Type == constants.WSFrameUnsubscribe {
			h.Unsubscribe(client, roomID)
			continue
		}

		// user can be removed from a room while their socket is open, and a
		// client must not keep receiving that room's events.
		if err := _authorizeSubscription(r, queries, roomID, userID); err != nil {
			_closeWebSocket(conn, websocket.StatusPolicyViolation, "not a participant of this room", reqID)
			return
		}

		if err := h.Subscribe(client, roomID); err != nil {
			log.Printf("websocket subscription refused: %v (req=%s)", err, reqID)
			_closeWebSocket(conn, websocket.StatusPolicyViolation, "subscription refused", reqID)
			return
		}
	}
}

// _authorizeSubscription reports whether userID may subscribe to roomID. Every
// failure is the same answer to the client, so a probe cannot distinguish a
// missing room from one the user is not a member of.
func _authorizeSubscription(r *http.Request, queries db.Querier, roomID, userID uuid.UUID) error {
	isParticipant, err := queries.IsParticipant(r.Context(), db.IsParticipantParams{
		RoomID: pgtype.UUID{Bytes: roomID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		log.Printf("websocket subscription check failed: %v (req=%s)", err, middleware.GetReqID(r.Context()))
		return err
	}
	if !isParticipant {
		return errNotParticipant
	}
	return nil
}

// _closeWebSocket sends a close frame explaining why the connection is being
// ended. The code is the contract the client reacts to, so it is never a
// generic failure: callers pick the status that matches the reason.
func _closeWebSocket(conn *websocket.Conn, code websocket.StatusCode, reason, reqID string) {
	if err := conn.Close(code, reason); err != nil {
		log.Printf("websocket close failed: %v (req=%s, code=%d)", err, reqID, code)
	}
}

// _publishEvent fans an event out to a room's subscribers. It is called after the
// originating write has already been committed, so a failure here is logged and
// dropped rather than surfaced to the HTTP caller: the write succeeded, and a
// client that missed the event refetches over REST on its next action.
func _publishEvent(r *http.Request, h *hub.Hub, roomID uuid.UUID, eventType string, data any) {
	payload, err := json.Marshal(realtimeEvent{Type: eventType, Data: data})
	if err != nil {
		log.Printf("publish %s failed: %v (req=%s)", eventType, err, middleware.GetReqID(r.Context()))
		return
	}
	h.Publish(roomID, payload)
}
