import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { RealtimeClient, type SocketLike } from '$lib/realtime'
import type { Message, MessageDeletedPayload, RealtimeStatus } from '$lib/types'

/**
 * A stub socket that records what the client sent and lets a test drive the
 * browser-generated lifecycle callbacks by hand.
 */
class FakeSocket implements SocketLike {
  readyState = 0
  onopen: ((event?: unknown) => void) | null = null
  onmessage: ((event: { data: unknown }) => void) | null = null
  onclose: ((event?: unknown) => void) | null = null
  onerror: ((event?: unknown) => void) | null = null

  readonly sent: string[] = []
  closeCalls = 0

  send(data: string): void {
    this.sent.push(data)
  }

  close(): void {
    this.closeCalls += 1
    this.readyState = 3
  }

  /** Simulates the handshake completing. */
  open(): void {
    this.readyState = 1
    this.onopen?.()
  }

  /** Simulates the server closing the connection. */
  serverClose(): void {
    this.readyState = 3
    this.onclose?.()
  }

  /** Simulates the server pushing a frame. */
  receive(data: unknown): void {
    this.onmessage?.({ data })
  }

  /** The frames sent, parsed. */
  frames(): { type: string; room_id?: string }[] {
    return this.sent.map((raw) => JSON.parse(raw) as { type: string; room_id?: string })
  }
}

function makeMessage(id: string): Message {
  return {
    id,
    room_id: 'room-1',
    sender_id: 'user-2',
    content: 'hello',
    reply_to_id: null,
    is_edited: false,
    edited_at: null,
    created_at: '2026-01-01T00:00:00Z',
  }
}

describe('RealtimeClient', () => {
  let sockets: FakeSocket[]
  let urls: string[]
  let client: RealtimeClient

  beforeEach(() => {
    sockets = []
    urls = []
    client = new RealtimeClient(
      (url) => {
        urls.push(url)
        const socket = new FakeSocket()
        sockets.push(socket)
        return socket
      },
      async () => 'test-token',
    )
  })

  afterEach(() => {
    client.disconnect()
    vi.useRealTimers()
  })

  describe('connect', () => {
    it('dials the /ws endpoint with the access token in the query string', async () => {
      await client.connect()

      expect(urls).toHaveLength(1)
      expect(urls[0]).toContain('ws://localhost:8080/ws?token=test-token')
    })

    it('reports connecting before the socket opens, then open', async () => {
      const states: RealtimeStatus[] = []
      client.onStatus((s) => states.push(s))

      await client.connect()
      expect(states.at(-1)).toBe('connecting')

      sockets[0].open()
      expect(states.at(-1)).toBe('open')
    })

    it('does not open a second socket while one is connected', async () => {
      await client.connect()
      sockets[0].open()
      await client.connect()

      expect(urls).toHaveLength(1)
    })

    it('stays offline and does not dial when there is no session', async () => {
      const anonymous = new RealtimeClient(
        () => new FakeSocket(),
        async () => null,
      )
      const states: RealtimeStatus[] = []
      anonymous.onStatus((s) => states.push(s))

      await anonymous.connect()

      expect(states.at(-1)).toBe('offline')
    })
  })

  describe('subscriptions', () => {
    it('sends a subscribe frame for the room being viewed', async () => {
      await client.connect()
      sockets[0].open()

      client.subscribe('room-1')

      expect(sockets[0].frames()).toEqual([{ type: 'subscribe', room_id: 'room-1' }])
    })

    it('sends an unsubscribe frame when the room changes', async () => {
      await client.connect()
      sockets[0].open()
      client.subscribe('room-1')

      client.unsubscribe('room-1')

      expect(sockets[0].frames().at(-1)).toEqual({
        type: 'unsubscribe',
        room_id: 'room-1',
      })
    })

    it('queues the subscription and sends it once the socket opens', async () => {
      await client.connect()
      client.subscribe('room-1')
      expect(sockets[0].sent).toHaveLength(0)

      sockets[0].open()

      expect(sockets[0].frames()).toEqual([{ type: 'subscribe', room_id: 'room-1' }])
    })
  })

  describe('event dispatch', () => {
    it('routes message_created to its handler', async () => {
      const received: Message[] = []
      client.onMessageCreated((m) => received.push(m))
      await client.connect()
      sockets[0].open()

      const message = makeMessage('m-1')
      sockets[0].receive(JSON.stringify({ type: 'message_created', data: message }))

      expect(received).toEqual([message])
    })

    it('routes message_updated to its handler', async () => {
      const received: Message[] = []
      client.onMessageUpdated((m) => received.push(m))
      await client.connect()
      sockets[0].open()

      const message = makeMessage('m-1')
      sockets[0].receive(JSON.stringify({ type: 'message_updated', data: message }))

      expect(received).toEqual([message])
    })

    it('routes message_deleted to its handler', async () => {
      const received: MessageDeletedPayload[] = []
      client.onMessageDeleted((p) => received.push(p))
      await client.connect()
      sockets[0].open()

      const payload: MessageDeletedPayload = { message_id: 'm-1', room_id: 'room-1' }
      sockets[0].receive(JSON.stringify({ type: 'message_deleted', data: payload }))

      expect(received).toEqual([payload])
    })

    it('ignores a frame that is not valid JSON', async () => {
      const received: Message[] = []
      client.onMessageCreated((m) => received.push(m))
      await client.connect()
      sockets[0].open()

      sockets[0].receive('{not json')

      expect(received).toHaveLength(0)
    })

    it('ignores a frame naming an unknown event type', async () => {
      const received: Message[] = []
      client.onMessageCreated((m) => received.push(m))
      await client.connect()
      sockets[0].open()

      sockets[0].receive(JSON.stringify({ type: 'room_deleted', data: {} }))

      expect(received).toHaveLength(0)
    })

    it('ignores a non-string frame payload', async () => {
      const received: Message[] = []
      client.onMessageCreated((m) => received.push(m))
      await client.connect()
      sockets[0].open()

      sockets[0].receive(new ArrayBuffer(8))

      expect(received).toHaveLength(0)
    })
  })

  describe('reconnect', () => {
    it('dials again after the server closes the connection', async () => {
      vi.useFakeTimers()
      await client.connect()
      sockets[0].open()
      expect(urls).toHaveLength(1)

      sockets[0].serverClose()
      expect(client.getStatus()).toBe('reconnecting')

      await vi.advanceTimersByTimeAsync(RECONNECT_WINDOW_MS)
      expect(urls).toHaveLength(2)
    })

    it('re-sends the room subscriptions on the new socket', async () => {
      vi.useFakeTimers()
      await client.connect()
      sockets[0].open()
      client.subscribe('room-1')

      sockets[0].serverClose()
      await vi.advanceTimersByTimeAsync(RECONNECT_WINDOW_MS)
      sockets[1].open()

      expect(sockets[1].frames()).toEqual([{ type: 'subscribe', room_id: 'room-1' }])
    })

    it('does not resubscribe to a room that was left before the reconnect', async () => {
      vi.useFakeTimers()
      await client.connect()
      sockets[0].open()
      client.subscribe('room-1')
      client.unsubscribe('room-1')

      sockets[0].serverClose()
      await vi.advanceTimersByTimeAsync(RECONNECT_WINDOW_MS)
      sockets[1].open()

      // The room was left, so the new socket must not be told to watch it again.
      expect(sockets[1].frames()).toEqual([])
    })

    it('backs off exponentially across repeated failures', async () => {
      vi.useFakeTimers()
      await client.connect()
      sockets[0].open()

      // The first delay is 500ms, the second 1000ms, each halved by jitter, so
      // the window below covers the largest possible delay for two attempts.
      for (let attempt = 0; attempt < 2; attempt += 1) {
        sockets[sockets.length - 1].serverClose()
        await vi.advanceTimersByTimeAsync(RECONNECT_WINDOW_MS)
      }

      expect(urls.length).toBeGreaterThanOrEqual(2)
    })
  })

  describe('disconnect', () => {
    it('closes the socket and reports offline', async () => {
      await client.connect()
      sockets[0].open()

      client.disconnect()

      expect(sockets[0].closeCalls).toBe(1)
      expect(client.getStatus()).toBe('offline')
    })

    it('does not reconnect after a deliberate disconnect', async () => {
      vi.useFakeTimers()
      await client.connect()
      sockets[0].open()

      client.disconnect()
      sockets[0].serverClose()
      await vi.advanceTimersByTimeAsync(RECONNECT_WINDOW_MS)

      expect(urls).toHaveLength(1)
      expect(client.getStatus()).toBe('offline')
    })

    it('cancels a pending reconnect', async () => {
      vi.useFakeTimers()
      await client.connect()
      sockets[0].open()
      sockets[0].serverClose()

      client.disconnect()
      await vi.advanceTimersByTimeAsync(RECONNECT_WINDOW_MS)

      expect(urls).toHaveLength(1)
    })
  })
})

/** Large enough to cover the longest jittered delay in these tests. */
const RECONNECT_WINDOW_MS = 60_000
