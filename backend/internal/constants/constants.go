package constants

type ctxKey string

const UserIDContextKey ctxKey = "user_id"

var USER_COLORS = []string{"--primary", "--secondary", "--success", "--warn", "--danger", "--info"}

// These names travel between the Go hub and
// frontend/src/lib/types.ts, so renaming one is a breaking change for every
const (
	// WebSocketPath is the single route the realtime transport is served on. It
	// is referenced by the router, by the log-redaction middleware, and by
	// frontend/src/lib/utils.ts
	WebSocketPath = "/ws"

	WSAuthQueryParam = "token"

	// WSFrameSubscribe and WSFrameUnsubscribe are the only frames a client may
	// send. They control delivery, they are not a write path: message writes
	// stay on the REST endpoints.
	WSFrameSubscribe   = "subscribe"
	WSFrameUnsubscribe = "unsubscribe"

	EventMessageCreated = "message_created"
	EventMessageUpdated = "message_updated"
	EventMessageDeleted = "message_deleted"
)

// Connect rate limit for the WebSocket handshake. A long-lived socket is
// counted once, at connect, so this bounds connection churn per IP rather than
// message traffic.
const WSConnectRateLimit = 60
