import { API_BASE_URL, api } from '$lib/api'
import { RECONNECT_BASE_DELAY_MS, RECONNECT_MAX_DELAY_MS } from '$lib/constants'
import type {
  ClientEvent,
  Message,
  MessageDeletedPayload,
  RealtimeStatus,
  ServerEvent,
  ServerEventType,
} from '$lib/types'
import { websocketUrl } from '$lib/utils'

/** Payload each server event carries. */
export type EventPayloads = {
  message_created: Message
  message_updated: Message
  message_deleted: MessageDeletedPayload
}

type Handlers = {
  [K in ServerEventType]: ((payload: EventPayloads[K]) => void) | null
}

/**
 * The slice of the WebSocket API this client uses. Depending on this rather than
 * on the DOM `WebSocket` is what lets the reconnect and dispatch logic be tested
 * in a plain Node environment with a stub socket.
 */
export type SocketLike = {
  readyState: number
  onopen: ((event: Event) => void) | null
  onmessage: ((event: MessageEvent) => void) | null
  onclose: ((event: CloseEvent) => void) | null
  onerror: ((event: Event) => void) | null
  send(data: string): void
  close(): void
}

/** The `readyState` of an open socket. Mirrors the DOM `WebSocket.OPEN`. */
const SOCKET_OPEN = 1

/**
 * The WebSocket transport. It owns the connection, its reconnect backoff, and
 * the set of rooms currently watched, and it dispatches server events to
 * registered handlers.
 *
 * The socket is an optimization, not a source of truth: a message that arrives
 * over REST is equally valid, and a client that misses an event refetches over
 * REST. That is why there is no replay, no sequence number, and no resync
 * protocol here.
 *
 * The socket factory and token source are injected rather than referenced
 * directly, so the reconnect and dispatch logic is testable without a real
 * server or a browser.
 */
export class RealtimeClient {
  private socket: SocketLike | null = null
  private createSocket: (url: string) => SocketLike
  private getToken: () => Promise<string | null>

  private handlers: Handlers = {
    message_created: null,
    message_updated: null,
    message_deleted: null,
  }

  private status: RealtimeStatus = 'offline'
  private statusHandler: ((status: RealtimeStatus) => void) | null = null

  /** Rooms to stay subscribed to across reconnects. */
  private watchedRooms = new Set<string>()
  private reconnectAttempts = 0
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private manuallyClosed = false

  constructor(createSocket: (url: string) => SocketLike, getToken: () => Promise<string | null>) {
    this.createSocket = createSocket
    this.getToken = getToken
  }

  getStatus(): RealtimeStatus {
    return this.status
  }

  onStatus(handler: (status: RealtimeStatus) => void): void {
    this.statusHandler = handler
    handler(this.status)
  }

  onMessageCreated(handler: (message: Message) => void): void {
    this.handlers.message_created = handler
  }

  onMessageUpdated(handler: (message: Message) => void): void {
    this.handlers.message_updated = handler
  }

  onMessageDeleted(handler: (payload: MessageDeletedPayload) => void): void {
    this.handlers.message_deleted = handler
  }

  /**
   * Opens the socket. A token is fetched first, so a session that has just
   * expired is renewed before the handshake rather than rejected by it.
   */
  async connect(): Promise<void> {
    if (this.socket) return
    this.manuallyClosed = false

    const token = await this.getToken()
    if (!token) {
      // No session: the page guard will redirect. Nothing to reconnect to.
      this.setStatus('offline')
      return
    }

    this.setStatus(this.reconnectAttempts > 0 ? 'reconnecting' : 'connecting')

    let socket: SocketLike
    try {
      socket = this.createSocket(websocketUrl(API_BASE_URL, token))
    } catch {
      // Constructing the socket can throw when the URL is malformed or the
      // browser blocks it. Retrying is the same path as a failed connection.
      this.setStatus('offline')
      this.scheduleReconnect()
      return
    }
    this.socket = socket

    socket.onopen = () => {
      this.reconnectAttempts = 0
      this.setStatus('open')
      // Restore the subscriptions the reconnect dropped.
      for (const roomId of this.watchedRooms) {
        this.send({ type: 'subscribe', room_id: roomId })
      }
    }

    socket.onmessage = (event) => {
      this.dispatch(event.data)
    }

    socket.onclose = () => {
      this.socket = null
      if (this.manuallyClosed) {
        this.setStatus('offline')
        return
      }
      this.setStatus('reconnecting')
      this.scheduleReconnect()
    }

    socket.onerror = () => {
      // The socket will close after an error, so the reconnect is scheduled
    }
  }

  disconnect(): void {
    this.manuallyClosed = true
    this.cancelReconnect()
    if (this.socket) {
      this.socket.close()
      this.socket = null
    }
    this.setStatus('offline')
  }

  subscribe(roomId: string): void {
    this.watchedRooms.add(roomId)
    this.send({ type: 'subscribe', room_id: roomId })
  }

  unsubscribe(roomId: string): void {
    this.watchedRooms.delete(roomId)
    this.send({ type: 'unsubscribe', room_id: roomId })
  }

  private send(event: ClientEvent): void {
    if (!this.socket || this.socket.readyState !== SOCKET_OPEN) return
    this.socket.send(JSON.stringify(event))
  }

  private dispatch(raw: unknown): void {
    if (typeof raw !== 'string') return

    let event: ServerEvent
    try {
      event = JSON.parse(raw) as ServerEvent
    } catch {
      return
    }
    if (!event || typeof event.type !== 'string' || !isServerEventType(event.type)) return

    this.handleEvent(event.type, event.data)
  }

  private handleEvent(type: ServerEventType, data: unknown): void {
    switch (type) {
      case 'message_created': {
        this.handlers.message_created?.(data as Message)
        return
      }
      case 'message_updated': {
        this.handlers.message_updated?.(data as Message)
        return
      }
      case 'message_deleted': {
        this.handlers.message_deleted?.(data as MessageDeletedPayload)
        return
      }
    }
  }

  private setStatus(status: RealtimeStatus): void {
    if (this.status === status) return
    this.status = status
    this.statusHandler?.(status)
  }

  private cancelReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  /**
   * Retries with exponential backoff and jitter. Jitter matters because a
   * backend restart disconnects every client at once, and an unjittered backoff
   * would have them all reconnect in the same instant.
   */
  private scheduleReconnect(): void {
    this.cancelReconnect()
    this.reconnectAttempts += 1

    const exponential = Math.min(
      RECONNECT_BASE_DELAY_MS * 2 ** (this.reconnectAttempts - 1),
      RECONNECT_MAX_DELAY_MS,
    )
    const delay = exponential / 2 + Math.random() * (exponential / 2)

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      void this.connect()
    }, delay)
  }
}

/**
 * The server event names this client understands. A frame naming anything else
 * is ignored rather than dispatched, so a future server-side event cannot
 * reach a handler as `undefined`.
 */
const SERVER_EVENT_TYPES: ReadonlySet<string> = new Set<ServerEventType>([
  'message_created',
  'message_updated',
  'message_deleted',
])

function isServerEventType(value: string): value is ServerEventType {
  return SERVER_EVENT_TYPES.has(value)
}

/**
 * The app's realtime client. The socket factory and the token source are
 * injected so the class itself carries no browser or network coupling.
 */
export const realtime = new RealtimeClient(
  (url) => new WebSocket(url),
  () => api.getValidAuthToken(),
)
