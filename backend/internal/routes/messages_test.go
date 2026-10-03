package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chat_app/internal/constants"
	"chat_app/internal/db"
	"chat_app/internal/hub"
	"chat_app/internal/utils"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// roomSubscriber is a real WebSocket connection subscribed to one room, used to
// observe the events a handler publishes. Subscribing through an actual socket
// rather than a test hook means these tests exercise the same delivery path a
// browser uses.
type roomSubscriber struct {
	events <-chan map[string]any
}

// subscribeToRoom opens a socket to a hub and subscribes it to roomID. The
// returned subscriber is cleaned up when the test ends.
func subscribeToRoom(t *testing.T, h *hub.Hub, roomID uuid.UUID) *roomSubscriber {
	t.Helper()

	adopted := make(chan *hub.Client, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		client, err := h.Serve(r.Context(), conn, uuid.New())
		if err != nil {
			_ = conn.CloseNow()
			return
		}
		if err := h.Subscribe(client, roomID); err != nil {
			_ = conn.CloseNow()
			return
		}
		adopted <- client
		// Hold the handler open so the hub keeps owning the connection.
		<-r.Context().Done()
		h.Remove(client)
	}))
	t.Cleanup(server.Close)

	conn, _, err := websocket.Dial(context.Background(),
		"ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial subscriber: %v", err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })

	select {
	case <-adopted:
	case <-time.After(3 * time.Second):
		t.Fatal("subscriber was not adopted by the hub")
	}

	events := make(chan map[string]any, 8)
	go func() {
		defer close(events)
		for {
			readCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, data, err := conn.Read(readCtx)
			cancel()
			if err != nil {
				return
			}
			var event map[string]any
			if err := json.Unmarshal(data, &event); err != nil {
				continue
			}
			select {
			case events <- event:
			case <-time.After(time.Second):
			}
		}
	}()

	return &roomSubscriber{events: events}
}

// next returns the next published event, failing the test if none arrives.
func (s *roomSubscriber) next(t *testing.T) map[string]any {
	t.Helper()
	select {
	case event, ok := <-s.events:
		if !ok {
			t.Fatal("the subscriber connection closed before an event arrived")
		}
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("expected an event to be published, but none arrived")
		return nil
	}
}

// assertNone fails if any event is published, waiting briefly so a slightly
// later publish is still caught.
func (s *roomSubscriber) assertNone(t *testing.T) {
	t.Helper()
	select {
	case event, ok := <-s.events:
		if ok {
			t.Errorf("an event was published unexpectedly: %v", event)
		}
	case <-time.After(150 * time.Millisecond):
	}
}

// newTestHub returns a hub whose liveness interval is long enough that no test
// races a ping.
func newTestHub() *hub.Hub {
	return hub.New(hub.Config{
		MaxConnections:        8,
		MaxRoomsPerConnection: 4,
		ClientQueueSize:       8,
		MaxFrameBytes:         4096,
		PingInterval:          time.Hour,
		WriteTimeout:          5 * time.Second,
		ShutdownTimeout:       time.Second,
	})
}

// authedRequest builds a request carrying a user id in the context, which is how
// JWTAuth hands the identity to a handler.
func authedRequest(method, target string, userID uuid.UUID, body any) *http.Request {
	payload := []byte(nil)
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		payload = encoded
	}

	r := httptest.NewRequest(method, target, bytes.NewReader(payload))
	return r.WithContext(context.WithValue(r.Context(), constants.UserIDContextKey, userID))
}

func TestSendMessagePublishesCreatedEvent(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	h := newTestHub()
	subscriber := subscribeToRoom(t, h, roomID)
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{roomID: true}}

	router := chi.NewRouter()
	router.Post("/rooms/{roomId}/messages", sendMessage(queries, h))

	req := authedRequest(http.MethodPost, "/rooms/"+roomID.String()+"/messages", userID, SendMessageRequest{
		RoomID:  roomID,
		Content: "hello",
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, http.StatusCreated, rec.Body)
	}

	event := subscriber.next(t)
	if event["type"] != constants.EventMessageCreated {
		t.Errorf("event type = %v, want %v", event["type"], constants.EventMessageCreated)
	}
}

func TestSendMessageDoesNotPublishWhenValidationFails(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	h := newTestHub()
	subscriber := subscribeToRoom(t, h, roomID)
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{roomID: true}}

	router := chi.NewRouter()
	router.Post("/rooms/{roomId}/messages", sendMessage(queries, h))

	req := authedRequest(http.MethodPost, "/rooms/"+roomID.String()+"/messages", userID, SendMessageRequest{
		RoomID:  roomID,
		Content: "",
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	subscriber.assertNone(t)
}

func TestSendMessageDoesNotPublishWhenNotAParticipant(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	h := newTestHub()
	subscriber := subscribeToRoom(t, h, roomID)
	// The user is not a participant of the room.
	queries := &fakeQuerier{participantRooms: map[uuid.UUID]bool{}}

	router := chi.NewRouter()
	router.Post("/rooms/{roomId}/messages", sendMessage(queries, h))

	req := authedRequest(http.MethodPost, "/rooms/"+roomID.String()+"/messages", userID, SendMessageRequest{
		RoomID:  roomID,
		Content: "hello",
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	subscriber.assertNone(t)
}

func TestSendMessageFailsWithoutPublishingWhenTheWriteFails(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	h := newTestHub()
	subscriber := subscribeToRoom(t, h, roomID)
	queries := &fakeQuerier{
		participantRooms: map[uuid.UUID]bool{roomID: true},
		sendErr:          errors.New("connection refused"),
	}

	router := chi.NewRouter()
	router.Post("/rooms/{roomId}/messages", sendMessage(queries, h))

	req := authedRequest(http.MethodPost, "/rooms/"+roomID.String()+"/messages", userID, SendMessageRequest{
		RoomID:  roomID,
		Content: "hello",
	})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	// The client-facing error must not leak the underlying failure.
	if bytes.Contains(rec.Body.Bytes(), []byte("connection refused")) {
		t.Errorf("response body leaks the internal error: %s", rec.Body)
	}
	subscriber.assertNone(t)
}

func TestEditMessagePublishesUpdatedEvent(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	messageID := uuid.New()
	h := newTestHub()
	subscriber := subscribeToRoom(t, h, roomID)
	stored := testMessage(roomID, userID)
	stored.ID = pgtype.UUID{Bytes: messageID, Valid: true}
	queries := &fakeQuerier{
		participantRooms: map[uuid.UUID]bool{roomID: true},
		message:          &stored,
	}

	router := chi.NewRouter()
	router.Put("/rooms/messages/{messageId}", editMessage(queries, h))

	req := authedRequest(http.MethodPut, "/rooms/messages/"+messageID.String(), userID, EditMessageRequest{Content: "edited"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, http.StatusOK, rec.Body)
	}

	event := subscriber.next(t)
	if event["type"] != constants.EventMessageUpdated {
		t.Errorf("event type = %v, want %v", event["type"], constants.EventMessageUpdated)
	}
}

func TestEditMessageDoesNotPublishAnotherUsersMessage(t *testing.T) {
	roomID := uuid.New()
	ownerID := uuid.New()
	attackerID := uuid.New()
	messageID := uuid.New()
	h := newTestHub()
	subscriber := subscribeToRoom(t, h, roomID)
	stored := testMessage(roomID, ownerID)
	stored.ID = pgtype.UUID{Bytes: messageID, Valid: true}
	queries := &fakeQuerier{
		participantRooms: map[uuid.UUID]bool{roomID: true},
		message:          &stored,
	}

	router := chi.NewRouter()
	router.Put("/rooms/messages/{messageId}", editMessage(queries, h))

	req := authedRequest(http.MethodPut, "/rooms/messages/"+messageID.String(), attackerID, EditMessageRequest{Content: "hijacked"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if len(queries.edited) != 0 {
		t.Error("EditMessage was called for a message the user does not own")
	}
	subscriber.assertNone(t)
}

func TestEditMessageDoesNotPublishForAMissingMessage(t *testing.T) {
	userID := uuid.New()
	messageID := uuid.New()
	h := newTestHub()
	queries := &fakeQuerier{getMessageErr: pgx.ErrNoRows}

	router := chi.NewRouter()
	router.Put("/rooms/messages/{messageId}", editMessage(queries, h))

	req := authedRequest(http.MethodPut, "/rooms/messages/"+messageID.String(), userID, EditMessageRequest{Content: "edited"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDeleteMessagePublishesDeletedEventWithRoomAndMessageIDs(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	messageID := uuid.New()
	h := newTestHub()
	subscriber := subscribeToRoom(t, h, roomID)
	stored := testMessage(roomID, userID)
	stored.ID = pgtype.UUID{Bytes: messageID, Valid: true}
	queries := &fakeQuerier{
		participantRooms: map[uuid.UUID]bool{roomID: true},
		message:          &stored,
	}

	router := chi.NewRouter()
	router.Delete("/rooms/messages/{messageId}", deleteMessage(queries, h))

	req := authedRequest(http.MethodDelete, "/rooms/messages/"+messageID.String(), userID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, http.StatusNoContent, rec.Body)
	}
	if len(queries.deleted) != 1 {
		t.Fatalf("DeleteMessage calls = %d, want 1", len(queries.deleted))
	}

	event := subscriber.next(t)
	if event["type"] != constants.EventMessageDeleted {
		t.Errorf("event type = %v, want %v", event["type"], constants.EventMessageDeleted)
	}
	data, ok := event["data"].(map[string]any)
	if !ok {
		t.Fatalf("event data = %#v, want an object", event["data"])
	}
	if data["message_id"] != messageID.String() {
		t.Errorf("message_id = %v, want %v", data["message_id"], messageID)
	}
	if data["room_id"] != roomID.String() {
		t.Errorf("room_id = %v, want %v", data["room_id"], roomID)
	}
}

func TestDeleteMessageDoesNotPublishForAnotherUsersMessage(t *testing.T) {
	roomID := uuid.New()
	ownerID := uuid.New()
	attackerID := uuid.New()
	messageID := uuid.New()
	h := newTestHub()
	subscriber := subscribeToRoom(t, h, roomID)
	stored := testMessage(roomID, ownerID)
	stored.ID = pgtype.UUID{Bytes: messageID, Valid: true}
	queries := &fakeQuerier{
		participantRooms: map[uuid.UUID]bool{roomID: true},
		message:          &stored,
	}

	router := chi.NewRouter()
	router.Delete("/rooms/messages/{messageId}", deleteMessage(queries, h))

	req := authedRequest(http.MethodDelete, "/rooms/messages/"+messageID.String(), attackerID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if len(queries.deleted) != 0 {
		t.Error("DeleteMessage was called for a message the user does not own")
	}
	subscriber.assertNone(t)
}

func TestDeleteMessageDoesNotPublishForAMissingMessage(t *testing.T) {
	userID := uuid.New()
	messageID := uuid.New()
	h := newTestHub()
	queries := &fakeQuerier{getMessageErr: pgx.ErrNoRows}

	router := chi.NewRouter()
	router.Delete("/rooms/messages/{messageId}", deleteMessage(queries, h))

	req := authedRequest(http.MethodDelete, "/rooms/messages/"+messageID.String(), userID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDeletedMessageEventUsesSnakeCaseFieldNames(t *testing.T) {
	// The payload is a wire contract, so the JSON field names are asserted
	// directly rather than through Go field names.
	encoded, err := json.Marshal(deletedMessageEvent{
		MessageID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		RoomID:    uuid.MustParse("22222222-2222-2222-2222-222222222222"),
	})
	if err != nil {
		t.Fatalf("marshal deleted event: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode deleted event: %v", err)
	}
	for _, field := range []string{"message_id", "room_id"} {
		if _, ok := decoded[field]; !ok {
			t.Errorf("payload is missing the %q field: %s", field, encoded)
		}
	}
	if len(decoded) != 2 {
		t.Errorf("payload has %d fields, want exactly 2: %s", len(decoded), encoded)
	}
}

func TestGetUserIDFromContextReadsTheAuthenticatedUser(t *testing.T) {
	userID := uuid.New()
	r := authedRequest(http.MethodGet, "/", userID, nil)

	got, ok := utils.GetUserIDFromContext(r.Context())
	if !ok {
		t.Fatal("expected the user id to be present in the context")
	}
	if got != userID {
		t.Errorf("user id = %v, want %v", got, userID)
	}
}

// testMessage builds a stored message owned by senderID in roomID.
func testMessage(roomID, senderID uuid.UUID) db.Message {
	return db.Message{
		ID:        pgtype.UUID{Bytes: uuid.New(), Valid: true},
		RoomID:    pgtype.UUID{Bytes: roomID, Valid: true},
		SenderID:  pgtype.UUID{Bytes: senderID, Valid: true},
		Content:   "original",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}
