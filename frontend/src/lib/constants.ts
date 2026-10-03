export const PUBLIC_ROUTES = ['/login', '/register']

/**
 * The path the realtime socket dials. Mirrors constants.WebSocketPath in
 * backend/internal/constants; the route is a wire contract (AGENTS.md 4.3).
 */
export const WEBSOCKET_PATH = '/ws'

/** Query parameter carrying the access token on the handshake. */
export const WEBSOCKET_TOKEN_PARAM = 'token'

export const COLOR_PALETTE = [
  '--primary',
  '--secondary',
  '--success',
  '--warn',
  '--danger',
  '--info',
]

/** Initial reconnect delay after the socket closes. */
export const RECONNECT_BASE_DELAY_MS = 500

/** Ceiling for the exponential reconnect backoff. */
export const RECONNECT_MAX_DELAY_MS = 30_000
