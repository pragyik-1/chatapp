package hub

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

var ErrTooManyRooms = errors.New("room subscription limit reached")

var ErrShuttingDown = errors.New("hub is shutting down")

type Client struct {
	conn   *websocket.Conn
	userID uuid.UUID
	out    chan []byte
	// done is closed by the writer goroutine on its way out, so send can stop
	// offering messages to a connection nobody is reading.
	done   chan struct{}
	cancel context.CancelFunc
	rooms  map[uuid.UUID]struct{}
}

// Hub owns the set of live connections and the room subscriptions over them.
// The zero value is not usable; construct one with New.
type Hub struct {
	cfg Config

	mu      sync.RWMutex
	rooms   map[uuid.UUID]map[*Client]struct{}
	clients map[*Client]struct{}
	closing bool

	// wg tracks writer goroutines so Shutdown can wait for them to drain.
	wg sync.WaitGroup
}

func (h *Hub) Subscribers(roomID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[roomID])
}

func New(cfg Config) *Hub {
	return &Hub{
		cfg:     cfg,
		rooms:   make(map[uuid.UUID]map[*Client]struct{}),
		clients: make(map[*Client]struct{}),
	}
}

// Serve registers conn as a client of userID and starts its writer goroutine.
// It applies the read limit and returns an error, leaving conn unregistered and
// still owned by the caller, when the hub is full or shutting down.
func (h *Hub) Serve(ctx context.Context, conn *websocket.Conn, userID uuid.UUID) (*Client, error) {
	h.mu.Lock()
	if h.closing {
		h.mu.Unlock()
		return nil, ErrShuttingDown
	}
	if len(h.clients) >= h.cfg.MaxConnections {
		h.mu.Unlock()
		return nil, errors.New("connection limit reached")
	}

	writerCtx, cancel := context.WithCancel(ctx)
	client := &Client{
		conn:   conn,
		userID: userID,
		out:    make(chan []byte, h.cfg.ClientQueueSize),
		done:   make(chan struct{}),
		cancel: cancel,
		rooms:  make(map[uuid.UUID]struct{}),
	}
	h.clients[client] = struct{}{}
	h.wg.Add(1)
	h.mu.Unlock()

	conn.SetReadLimit(h.cfg.MaxFrameBytes)
	go func() {
		defer h.wg.Done()
		client.writeLoop(writerCtx, h.cfg)
	}()

	return client, nil
}

// Publish delivers an already-marshalled payload to every client subscribed to
// roomID. Delivery is non-blocking: a client whose queue is full is a slow
// consumer, so it is dropped from the hub and the client will reconnect and
// refetch. Dropping is safe because the socket is a hint; REST stays the source
// of truth.
func (h *Hub) Publish(roomID uuid.UUID, payload []byte) {
	h.mu.RLock()
	subscribers := make([]*Client, 0, len(h.rooms[roomID]))
	for client := range h.rooms[roomID] {
		subscribers = append(subscribers, client)
	}
	h.mu.RUnlock()

	var slow []*Client
	for _, client := range subscribers {
		if !client.send(payload) {
			slow = append(slow, client)
		}
	}
	for _, client := range slow {
		log.Printf("closed slow websocket client: outbound queue full (user=%s)", client.userID)
		h.Remove(client)
	}
}

// Subscribe adds client to roomID's delivery set. It returns an error when the
// client has reached MaxRoomsPerConnection or the hub is shutting down.
func (h *Hub) Subscribe(client *Client, roomID uuid.UUID) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closing {
		return ErrShuttingDown
	}
	if _, subscribed := client.rooms[roomID]; subscribed {
		return nil
	}
	if len(client.rooms) >= h.cfg.MaxRoomsPerConnection {
		return ErrTooManyRooms
	}

	client.rooms[roomID] = struct{}{}
	if _, ok := h.rooms[roomID]; !ok {
		h.rooms[roomID] = make(map[*Client]struct{})
	}
	h.rooms[roomID][client] = struct{}{}
	return nil
}

// Unsubscribe removes client from roomID. It is idempotent.
func (h *Hub) Unsubscribe(client *Client, roomID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.detach(client, roomID)
}

// Remove deregisters client and tears down its connection. It is idempotent:
// only the first call for a given client performs the teardown.
func (h *Hub) Remove(client *Client) {
	h.mu.Lock()
	if _, registered := h.clients[client]; !registered {
		h.mu.Unlock()
		return
	}
	delete(h.clients, client)
	for roomID := range client.rooms {
		h.detach(client, roomID)
	}
	h.mu.Unlock()

	// Cancel first so the writer goroutine's select unblocks immediately, then
	// hard-close the socket. A graceful close is deliberately not used here: it
	// waits for the peer's close frame, and Remove runs on hot paths such as
	// Publish, where blocking on a misbehaving client is not acceptable.
	client.cancel()
	// CloseNow is best-effort: the connection is frequently already gone, since
	// the usual path is a read error on the client side tearing the socket
	// down. That is the expected outcome, not a fault worth logging.
	if err := client.conn.CloseNow(); err != nil && !errors.Is(err, net.ErrClosed) {
		log.Printf("close websocket connection failed: %v (user=%s)", err, client.userID)
	}
}

// Shutdown closes every client and waits for the writers to drain, or for ctx
// to expire. It refuses new connections once called.
func (h *Hub) Shutdown(ctx context.Context) {
	h.mu.Lock()
	h.closing = true
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.Unlock()

	for _, client := range clients {
		h.Remove(client)
	}

	drained := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(drained)
	}()

	select {
	case <-drained:
	case <-ctx.Done():
		log.Printf("websocket shutdown timed out with writers still running: %v", ctx.Err())
	}
}

// detach removes client from one room, dropping the room entry when it empties.
// Callers must hold h.mu.
func (h *Hub) detach(client *Client, roomID uuid.UUID) {
	if _, subscribed := client.rooms[roomID]; !subscribed {
		return
	}
	delete(client.rooms, roomID)
	delete(h.rooms[roomID], client)
	if len(h.rooms[roomID]) == 0 {
		delete(h.rooms, roomID)
	}
}

// send offers payload to the writer goroutine. It reports false when the
// writer has already exited or the queue is full.
func (c *Client) send(payload []byte) bool {
	select {
	case <-c.done:
		return false
	default:
	}

	select {
	case c.out <- payload:
		return true
	default:
		return false
	}
}

// writeLoop is the only writer on the connection. It drains the outbound queue,
// pings on an interval.
func (c *Client) writeLoop(ctx context.Context, cfg Config) {
	defer close(c.done)

	ticker := time.NewTicker(cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case payload := <-c.out:
			if err := c.write(ctx, cfg.WriteTimeout, func(writeCtx context.Context) error {
				return c.conn.Write(writeCtx, websocket.MessageText, payload)
			}); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(ctx, cfg.WriteTimeout, c.conn.Ping); err != nil {
				return
			}
		}
	}
}

// write runs fn under a write timeout derived from ctx. A ping or write that
// exceeds the timeout fails the connection: a client that cannot accept a frame
// within WriteTimeout is not usable.
func (c *Client) write(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	writeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return fn(writeCtx)
}
