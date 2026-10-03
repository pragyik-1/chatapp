package hub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

const readTimeout = 3 * time.Second

// testConfig returns a hub config sized for tests, with a liveness interval long
// enough that no test races a ping.
func testConfig() Config {
	return Config{
		MaxConnections:        4,
		MaxRoomsPerConnection: 2,
		ClientQueueSize:       2,
		MaxFrameBytes:         4096,
		PingInterval:          time.Hour,
		WriteTimeout:          readTimeout,
		ShutdownTimeout:       time.Second,
	}
}

// connectedClient is a live client backed by a real WebSocket pair. Events are
// read on dialed, the client side of the socket, because that is where a real
// browser would receive them.
type connectedClient struct {
	client *Client
	dialed *websocket.Conn
}

// newTestHubServer serves a bare /ws endpoint whose handler adopts every
// connection into a hub, so tests exercise the real writer goroutine and the real
// socket rather than a stub.
func newTestHubServer(t *testing.T, h *Hub) (*httptest.Server, <-chan *connectedClient) {
	t.Helper()
	adopted := make(chan *connectedClient, 8)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		client, err := h.Serve(r.Context(), conn, uuid.New())
		if err != nil {
			_ = conn.CloseNow()
			return
		}
		adopted <- &connectedClient{client: client, dialed: conn}
		// Hold the request open so the hub owns the connection for the test.
		<-r.Context().Done()
		h.Remove(client)
	}))
	t.Cleanup(ts.Close)

	return ts, adopted
}

// connect dials the test server and returns the adopted client.
func connect(t *testing.T, ts *httptest.Server, adopted <-chan *connectedClient) *connectedClient {
	t.Helper()
	conn, _, err := websocket.Dial(dialContext(t), "ws"+strings.TrimPrefix(ts.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })

	select {
	case c := <-adopted:
		c.dialed = conn
		return c
	case <-time.After(readTimeout):
		t.Fatal("server did not adopt the connection")
		return nil
	}
}

func dialContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	t.Cleanup(cancel)
	return ctx
}

func TestPublishReachesEverySubscriberOfTheRoom(t *testing.T) {
	h := New(testConfig())
	ts, adopted := newTestHubServer(t, h)
	roomID := uuid.New()

	first := connect(t, ts, adopted)
	second := connect(t, ts, adopted)
	for _, c := range []*connectedClient{first, second} {
		if err := h.Subscribe(c.client, roomID); err != nil {
			t.Fatalf("subscribe: %v", err)
		}
	}

	h.Publish(roomID, []byte(`{"type":"message_created"}`))

	for i, c := range []*connectedClient{first, second} {
		if got := readFrame(t, c.dialed); got != `{"type":"message_created"}` {
			t.Errorf("client %d received %q, want the published payload", i, got)
		}
	}
}

func TestPublishDoesNotReachSubscribersOfAnotherRoom(t *testing.T) {
	h := New(testConfig())
	ts, adopted := newTestHubServer(t, h)

	watched := connect(t, ts, adopted)
	other := connect(t, ts, adopted)
	if err := h.Subscribe(watched.client, uuid.New()); err != nil {
		t.Fatalf("subscribe watched: %v", err)
	}
	if err := h.Subscribe(other.client, uuid.New()); err != nil {
		t.Fatalf("subscribe other: %v", err)
	}

	h.Publish(uuid.New(), []byte("nobody is listening"))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, _, err := watched.dialed.Read(ctx); err == nil {
		t.Fatal("a client subscribed to a different room received an event")
	}
}

func TestPublishToRoomWithNoSubscribersIsNotAnError(t *testing.T) {
	h := New(testConfig())

	// Must not panic or block on an unknown room.
	h.Publish(uuid.New(), []byte("dropped"))
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	h := New(testConfig())
	ts, adopted := newTestHubServer(t, h)
	roomID := uuid.New()

	c := connect(t, ts, adopted)
	if err := h.Subscribe(c.client, roomID); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	h.Unsubscribe(c.client, roomID)
	h.Publish(roomID, []byte("after unsubscribe"))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, _, err := c.dialed.Read(ctx); err == nil {
		t.Fatal("received an event after unsubscribing")
	}
}

func TestSubscribeIsIdempotent(t *testing.T) {
	h := New(testConfig())
	ts, adopted := newTestHubServer(t, h)
	roomID := uuid.New()

	c := connect(t, ts, adopted)
	for range 2 {
		if err := h.Subscribe(c.client, roomID); err != nil {
			t.Fatalf("subscribe: %v", err)
		}
	}

	if got := h.Subscribers(roomID); got != 1 {
		t.Errorf("subscribers = %d, want 1 after subscribing twice", got)
	}
}

func TestSubscribeRejectsMoreRoomsThanTheLimit(t *testing.T) {
	cfg := testConfig()
	cfg.MaxRoomsPerConnection = 2
	h := New(cfg)
	ts, adopted := newTestHubServer(t, h)

	c := connect(t, ts, adopted)
	if err := h.Subscribe(c.client, uuid.New()); err != nil {
		t.Fatalf("first subscribe: %v", err)
	}
	if err := h.Subscribe(c.client, uuid.New()); err != nil {
		t.Fatalf("second subscribe: %v", err)
	}

	err := h.Subscribe(c.client, uuid.New())
	if err == nil {
		t.Fatal("expected the third subscribe to exceed the limit")
	}
	if err != ErrTooManyRooms {
		t.Errorf("error = %v, want %v", err, ErrTooManyRooms)
	}
}

func TestServeRejectsConnectionsBeyondTheLimit(t *testing.T) {
	cfg := testConfig()
	cfg.MaxConnections = 1
	h := New(cfg)
	ts, adopted := newTestHubServer(t, h)

	connect(t, ts, adopted) // occupies the only slot

	// The handler closes the refused connection, so the dial succeeds at the
	// transport level only if the upgrade is refused: a refusal arrives as a
	// close, which the read below observes.
	conn, _, err := websocket.Dial(dialContext(t), "ws"+strings.TrimPrefix(ts.URL, "http"), nil)
	if err != nil {
		return // rejected during the upgrade, which is also acceptable
	}
	defer conn.CloseNow()

	if _, _, err := conn.Read(dialContext(t)); err == nil {
		t.Fatal("expected a refused connection to be closed")
	}
}

func TestPublishClosesASlowConsumerAndKeepsServingOthers(t *testing.T) {
	cfg := testConfig()
	cfg.ClientQueueSize = 1
	h := New(cfg)
	ts, adopted := newTestHubServer(t, h)
	roomID := uuid.New()

	// A client that never reads fills its queue. Stopping its writer from
	// draining is what a stuck client looks like to the hub.
	slow := connect(t, ts, adopted)
	healthy := connect(t, ts, adopted)
	if err := h.Subscribe(slow.client, roomID); err != nil {
		t.Fatalf("subscribe slow: %v", err)
	}
	if err := h.Subscribe(healthy.client, roomID); err != nil {
		t.Fatalf("subscribe healthy: %v", err)
	}

	// Publish enough to overflow the one-slot queue. The slow client is
	// dropped once its queue is full; the healthy one keeps receiving.
	var lastForHealthy string
	for i := range 5 {
		payload := []byte{byte('a' + i)}
		h.Publish(roomID, payload)
		if got := readFrame(t, healthy.dialed); len(got) > 0 {
			lastForHealthy = string(got)
		}
	}

	if lastForHealthy == "" {
		t.Error("the healthy subscriber stopped receiving events")
	}
	if got := h.Subscribers(roomID); got > 2 {
		t.Errorf("subscribers = %d, want at most 2", got)
	}
}

func TestRemoveIsIdempotent(t *testing.T) {
	h := New(testConfig())
	ts, adopted := newTestHubServer(t, h)

	c := connect(t, ts, adopted)
	h.Remove(c.client)
	// A second removal must be a no-op rather than a double close or a panic.
	h.Remove(c.client)
}

func TestRemoveDropsRoomSubscriptions(t *testing.T) {
	h := New(testConfig())
	ts, adopted := newTestHubServer(t, h)
	roomID := uuid.New()

	c := connect(t, ts, adopted)
	if err := h.Subscribe(c.client, roomID); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	h.Remove(c.client)

	if got := h.Subscribers(roomID); got != 0 {
		t.Errorf("subscribers = %d, want 0 after the client was removed", got)
	}
}

func TestShutdownClosesEveryClient(t *testing.T) {
	h := New(testConfig())
	ts, adopted := newTestHubServer(t, h)

	first := connect(t, ts, adopted)
	second := connect(t, ts, adopted)

	h.Shutdown(context.Background())

	for i, c := range []*connectedClient{first, second} {
		if _, _, err := c.dialed.Read(dialContext(t)); err == nil {
			t.Errorf("client %d was still open after shutdown", i)
		}
	}
}

func TestServeRejectsNewConnectionsAfterShutdown(t *testing.T) {
	h := New(testConfig())
	ts, adopted := newTestHubServer(t, h)

	h.Shutdown(context.Background())

	conn, _, err := websocket.Dial(dialContext(t), "ws"+strings.TrimPrefix(ts.URL, "http"), nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	if _, _, err := conn.Read(dialContext(t)); err == nil {
		t.Fatal("a connection was accepted after shutdown began")
	}
	_ = adopted
}

func TestPublishAfterShutdownDoesNotPanic(t *testing.T) {
	h := New(testConfig())
	h.Shutdown(context.Background())

	h.Publish(uuid.New(), []byte("after shutdown"))
}

func TestConfigFromEnvFallsBackToDefaults(t *testing.T) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}
	if cfg != defaultConfig() {
		t.Errorf("config = %+v, want the defaults %+v", cfg, defaultConfig())
	}
}

func TestConfigFromEnvReadsOverrides(t *testing.T) {
	t.Setenv(envMaxConnections, "7")
	t.Setenv(envPingInterval, "45s")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}
	if cfg.MaxConnections != 7 {
		t.Errorf("MaxConnections = %d, want 7", cfg.MaxConnections)
	}
	if cfg.PingInterval != 45*time.Second {
		t.Errorf("PingInterval = %s, want 45s", cfg.PingInterval)
	}
	// Untouched variables keep their defaults.
	if cfg.ClientQueueSize != defaultConfig().ClientQueueSize {
		t.Errorf("ClientQueueSize = %d, want the default %d", cfg.ClientQueueSize, defaultConfig().ClientQueueSize)
	}
}

func TestConfigFromEnvRejectsInvalidValues(t *testing.T) {
	for _, tc := range []struct{ name, value string }{
		{envMaxConnections, "not-a-number"},
		{envMaxConnections, "0"},
		{envMaxConnections, "-1"},
		{envMaxRoomsPerConnection, "many"},
		{envClientQueueSize, "0"},
		{envMaxFrameBytes, "-8"},
		{envPingInterval, "soon"},
		{envPingInterval, "0s"},
		{envWriteTimeout, "-1s"},
		{envShutdownTimeout, "nope"},
	} {
		t.Run(tc.name+"="+tc.value, func(t *testing.T) {
			t.Setenv(tc.name, tc.value)
			if _, err := ConfigFromEnv(); err == nil {
				t.Fatalf("expected %s=%q to be rejected", tc.name, tc.value)
			}
		})
	}
}

// readFrame reads one text frame from conn.
func readFrame(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	return string(data)
}
