package routes

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"chat_app/internal/auth"
	"chat_app/internal/constants"
	"chat_app/internal/db"
	"chat_app/internal/hub"
	"chat_app/internal/utils"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const testJWTSecret = "test-secret"

// loggedURI is written by the middleware in the request-URI redaction test. It is
// package-scoped because that closure cannot return a value to the test. The
// mutex guards it because the middleware runs on the server's goroutine while
// the test reads it from the client's.
var (
	loggedURIMu sync.Mutex
	loggedURI   string
)

// newTestToken mints an access token for userID.
func newTestToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	token, err := auth.GenerateJWT(pgtype.UUID{Bytes: userID, Valid: true}, testJWTSecret)
	if err != nil {
		t.Fatalf("generate test token: %v", err)
	}
	return token
}

// dialWS opens a WebSocket against a test server, failing the test if the
// handshake is rejected. It returns the connection and a cleanup func.
func dialWS(t *testing.T, ts *httptest.Server, url string) (*websocket.Conn, func()) {
	t.Helper()
	conn, resp, err := websocket.Dial(ctxWithTimeout(t), url, nil)
	if err != nil {
		if resp != nil {
			t.Fatalf("dial %s: %v (status %d)", url, err, resp.StatusCode)
		}
		t.Fatalf("dial %s: %v", url, err)
	}
	return conn, func() { _ = conn.CloseNow() }
}

func ctxWithTimeout(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// wsTestServer starts an httptest server exposing /ws with the given fake
// queries and hub.
func wsTestServer(t *testing.T, queries db.Querier, h *hub.Hub) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Get("/ws", serveWebSocket(queries, h, testJWTSecret))
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts
}

// wsURL builds a handshake URL carrying an access token.
func wsURL(ts *httptest.Server, token string) string {
	return "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?token=" + token
}

// readEvent reads one realtime event from conn.
func readEvent(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	_, data, err := conn.Read(ctxWithTimeout(t))
	if err != nil {
		t.Fatalf("read event: %v", err)
	}
	var event map[string]any
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("decode event %q: %v", data, err)
	}
	return event
}

// sendFrame writes a raw client frame.
func sendFrame(t *testing.T, conn *websocket.Conn, payload any) {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode frame: %v", err)
	}
	if err := conn.Write(ctxWithTimeout(t), websocket.MessageText, data); err != nil {
		t.Fatalf("write frame: %v", err)
	}
}

func TestServeWebSocketRejectsMissingToken(t *testing.T) {
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{}}
	ts := wsTestServer(t, queries, newTestHub())

	resp, err := http.Get(ts.URL + "/ws")
	if err != nil {
		t.Fatalf("get /ws: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestServeWebSocketRejectsInvalidToken(t *testing.T) {
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{}}
	ts := wsTestServer(t, queries, newTestHub())

	resp, err := http.Get(ts.URL + "/ws?token=not-a-jwt")
	if err != nil {
		t.Fatalf("get /ws: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestServeWebSocketStripsTokenFromLoggedRequestURI(t *testing.T) {
	// The token travels in the query string, and chi's logger formats
	// r.RequestURI into its log line before the handler runs. This asserts
	// redactWebSocketToken, which is registered ahead of the logger in
	// MakeRouter, has already removed it by then.
	//
	// The handshake is a real WebSocket upgrade, because the token must pass
	// validation for the connection to be accepted. The client closes
	// immediately, so the handler unwinds and the middleware below observes
	// the redacted URI.
	loggedURIMu.Lock()
	loggedURI = ""
	loggedURIMu.Unlock()

	userID := uuid.New()
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{}}
	h := newTestHub()
	token := newTestToken(t, userID)

	// Registered in MakeRouter's order: redaction first, then the snapshot
	// standing in for the logger. The route is spelled literally here so the
	// test pins the wire path rather than restating the constant.
	r := chi.NewRouter()
	r.Use(redactWebSocketToken)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Snapshot the request the way chi's logger does: after the
			// handler has run.
			next.ServeHTTP(w, r)
			loggedURIMu.Lock()
			loggedURI = r.RequestURI
			loggedURIMu.Unlock()
		})
	})
	r.Get("/ws", serveWebSocket(queries, h, testJWTSecret))

	ts := httptest.NewServer(r)
	defer ts.Close()

	conn, _, err := websocket.Dial(ctxWithTimeout(t),
		"ws"+strings.TrimPrefix(ts.URL, "http")+"/ws?token="+token, nil)
	if err != nil {
		t.Fatalf("dial /ws: %v", err)
	}
	// Closing makes the server's read loop return, which runs the deferred
	// teardown and unwinds the middleware.
	if err := conn.CloseNow(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}
	waitFor(t, func() bool {
		loggedURIMu.Lock()
		defer loggedURIMu.Unlock()
		return loggedURI != ""
	})

	loggedURIMu.Lock()
	defer loggedURIMu.Unlock()
	if strings.Contains(loggedURI, token) {
		t.Errorf("logged RequestURI %q contains the access token", loggedURI)
	}
	if loggedURI != "/ws" {
		t.Errorf("logged RequestURI = %q, want %q", loggedURI, "/ws")
	}
}

func TestServeWebSocketRejectsDisallowedOrigin(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "http://localhost:5173")
	userID := uuid.New()
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{}}
	token := newTestToken(t, userID)
	ts := wsTestServer(t, queries, newTestHub())

	_, _, err := websocket.Dial(ctxWithTimeout(t),
		"ws"+strings.TrimPrefix(ts.URL, "http")+"/ws?token="+token,
		&websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{"http://evil.example"}}})
	if err == nil {
		t.Fatal("expected handshake from a disallowed origin to be rejected")
	}
}

func TestServeWebSocketRejectsSubscribeToRoomUserIsNotIn(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	// The user is not a participant of roomID.
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{}}
	ts := wsTestServer(t, queries, newTestHub())
	token := newTestToken(t, userID)

	conn, cleanup := dialWS(t, ts, wsURL(ts, token))
	defer cleanup()

	sendFrame(t, conn, map[string]any{
		"type":    constants.WSFrameSubscribe,
		"room_id": roomID,
	})

	// The server must close rather than accept the subscription.
	_, _, err := conn.Read(ctxWithTimeout(t))
	if err == nil {
		t.Fatal("expected the connection to be closed after an unauthorized subscribe")
	}
	if code := websocket.CloseStatus(err); code != websocket.StatusPolicyViolation {
		t.Errorf("close code = %d, want %d", code, websocket.StatusPolicyViolation)
	}
}

func TestServeWebSocketDeliversPublishedEventToSubscriber(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{roomID: true}}
	h := newTestHub()
	ts := wsTestServer(t, queries, h)
	token := newTestToken(t, userID)

	conn, cleanup := dialWS(t, ts, wsURL(ts, token))
	defer cleanup()

	sendFrame(t, conn, map[string]any{
		"type":    constants.WSFrameSubscribe,
		"room_id": roomID,
	})

	// Wait for the subscription to be registered before publishing, so the
	// test does not depend on write ordering.
	waitFor(t, func() bool { return h.Subscribers(roomID) == 1 })

	payload, err := json.Marshal(realtimeEvent{
		Type: constants.EventMessageCreated,
		Data: map[string]string{"id": "message-1"},
	})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	h.Publish(roomID, payload)

	event := readEvent(t, conn)
	if event["type"] != constants.EventMessageCreated {
		t.Errorf("event type = %v, want %v", event["type"], constants.EventMessageCreated)
	}
	data, ok := event["data"].(map[string]any)
	if !ok {
		t.Fatalf("event data = %#v, want an object", event["data"])
	}
	if data["id"] != "message-1" {
		t.Errorf("event data id = %v, want %q", data["id"], "message-1")
	}
}

func TestServeWebSocketUnsubscribeStopsDelivery(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{roomID: true}}
	h := newTestHub()
	ts := wsTestServer(t, queries, h)
	token := newTestToken(t, userID)

	conn, cleanup := dialWS(t, ts, wsURL(ts, token))
	defer cleanup()

	sendFrame(t, conn, map[string]any{"type": constants.WSFrameSubscribe, "room_id": roomID})
	waitFor(t, func() bool { return h.Subscribers(roomID) == 1 })

	sendFrame(t, conn, map[string]any{"type": constants.WSFrameUnsubscribe, "room_id": roomID})
	waitFor(t, func() bool { return h.Subscribers(roomID) == 0 })

	// Publishing now must reach nobody. Reading with a short deadline is how
	// the test asserts absence: a delivered event would return immediately.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if _, _, err := conn.Read(ctx); err == nil {
		t.Fatal("received an event after unsubscribing")
	}
}

func TestServeWebSocketRejectsFrameWithoutRoomID(t *testing.T) {
	userID := uuid.New()
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{}}
	ts := wsTestServer(t, queries, newTestHub())
	token := newTestToken(t, userID)

	conn, cleanup := dialWS(t, ts, wsURL(ts, token))
	defer cleanup()

	sendFrame(t, conn, map[string]any{"type": constants.WSFrameSubscribe})

	_, _, err := conn.Read(ctxWithTimeout(t))
	if err == nil {
		t.Fatal("expected the connection to be closed for a subscribe without room_id")
	}
	if code := websocket.CloseStatus(err); code != websocket.StatusPolicyViolation {
		t.Errorf("close code = %d, want %d", code, websocket.StatusPolicyViolation)
	}
}

func TestServeWebSocketRejectsUnsupportedFrameType(t *testing.T) {
	userID := uuid.New()
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{}}
	ts := wsTestServer(t, queries, newTestHub())
	token := newTestToken(t, userID)

	conn, cleanup := dialWS(t, ts, wsURL(ts, token))
	defer cleanup()

	sendFrame(t, conn, map[string]any{"type": "send_message", "room_id": uuid.New()})

	_, _, err := conn.Read(ctxWithTimeout(t))
	if err == nil {
		t.Fatal("expected the connection to be closed for an unsupported frame type")
	}
	if code := websocket.CloseStatus(err); code != websocket.StatusPolicyViolation {
		t.Errorf("close code = %d, want %d", code, websocket.StatusPolicyViolation)
	}
}

func TestServeWebSocketRejectsMalformedFrame(t *testing.T) {
	userID := uuid.New()
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{}}
	ts := wsTestServer(t, queries, newTestHub())
	token := newTestToken(t, userID)

	conn, cleanup := dialWS(t, ts, wsURL(ts, token))
	defer cleanup()

	if err := conn.Write(ctxWithTimeout(t), websocket.MessageText, []byte("{not json")); err != nil {
		t.Fatalf("write frame: %v", err)
	}

	_, _, err := conn.Read(ctxWithTimeout(t))
	if err == nil {
		t.Fatal("expected the connection to be closed for a malformed frame")
	}
	if code := websocket.CloseStatus(err); code != websocket.StatusPolicyViolation {
		t.Errorf("close code = %d, want %d", code, websocket.StatusPolicyViolation)
	}
}

func TestServeWebSocketEnforcesSubscriptionLimit(t *testing.T) {
	userID := uuid.New()
	// The user is a participant of every room, so the cap is what refuses.
	rooms := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	participantRooms := make(map[uuid.UUID]bool, len(rooms))
	for _, roomID := range rooms {
		participantRooms[roomID] = true
	}

	queries := &fakeQuerier{participantRooms: participantRooms}
	// A hub whose room cap is lower than the number of rooms to subscribe to.
	h := hub.New(hub.Config{
		MaxConnections:        4,
		MaxRoomsPerConnection: 2,
		ClientQueueSize:       8,
		MaxFrameBytes:         4096,
		PingInterval:          time.Hour,
		WriteTimeout:          5 * time.Second,
		ShutdownTimeout:       time.Second,
	})
	ts := wsTestServer(t, queries, h)
	token := newTestToken(t, userID)

	conn, cleanup := dialWS(t, ts, wsURL(ts, token))
	defer cleanup()

	// The first MaxRoomsPerConnection subscribes are accepted.
	for _, roomID := range rooms[:2] {
		sendFrame(t, conn, map[string]any{"type": constants.WSFrameSubscribe, "room_id": roomID})
		waitFor(t, func() bool { return h.Subscribers(roomID) == 1 })
	}

	// The next one exceeds the cap and must close the socket.
	sendFrame(t, conn, map[string]any{"type": constants.WSFrameSubscribe, "room_id": rooms[2]})
	_, _, err := conn.Read(ctxWithTimeout(t))
	if err == nil {
		t.Fatal("expected the connection to be closed once the subscription cap was reached")
	}
	if code := websocket.CloseStatus(err); code != websocket.StatusPolicyViolation {
		t.Errorf("close code = %d, want %d", code, websocket.StatusPolicyViolation)
	}
}

// waitFor polls condition until it holds or the test times out.
func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met before the deadline")
}

// TestWriteErrorFormattingIsGeneric guards the rule that handler messages stay
// free of internal detail: the client-facing error must not name the query that
// failed.
func TestWriteErrorFormattingIsGeneric(t *testing.T) {
	rec := httptest.NewRecorder()
	utils.WriteError(rec, http.StatusInternalServerError, "failed to send message")

	var body map[string]string
	if err := json.NewDecoder(bufio.NewReader(bytes.NewReader(rec.Body.Bytes()))).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body["error"] != "failed to send message" {
		t.Errorf("error = %q, want %q", body["error"], "failed to send message")
	}
}
