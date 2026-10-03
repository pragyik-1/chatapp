import { describe, expect, it } from 'vitest'
import { upsertMessage, websocketUrl } from '$lib/utils'
import type { Message } from '$lib/types'

function message(id: string, content: string, roomId = 'room-1'): Message {
  return {
    id,
    room_id: roomId,
    sender_id: 'user-1',
    content,
    reply_to_id: null,
    is_edited: false,
    edited_at: null,
    created_at: '2026-01-01T00:00:00Z',
  }
}

describe('websocketUrl', () => {
  it('derives a ws:// URL from an http:// base and carries the token', () => {
    expect(websocketUrl('http://localhost:8080', 'abc.def.ghi')).toBe(
      'ws://localhost:8080/ws?token=abc.def.ghi',
    )
  })

  it('derives a wss:// URL from an https:// base', () => {
    expect(websocketUrl('https://api.example.com', 'tok')).toBe(
      'wss://api.example.com/ws?token=tok',
    )
  })

  it('percent-encodes a token containing URL-significant characters', () => {
    const url = new URL(websocketUrl('http://localhost:8080', 'a+b/c=d'))
    expect(url.searchParams.get('token')).toBe('a+b/c=d')
  })
})

describe('upsertMessage', () => {
  it('appends a message that is not yet in the list', () => {
    const existing = [message('1', 'first')]
    const next = upsertMessage(existing, message('2', 'second'))

    expect(next).toHaveLength(2)
    expect(next.map((m) => m.id)).toEqual(['1', '2'])
  })

  it('replaces an existing message in place, keeping its position', () => {
    const existing = [message('1', 'first'), message('2', 'second')]
    const next = upsertMessage(existing, message('2', 'edited', 'room-1'))

    expect(next).toHaveLength(2)
    expect(next[1].content).toBe('edited')
  })

  it('returns the same array when the message is already identical', () => {
    const existing = [message('1', 'first')]
    expect(upsertMessage(existing, existing[0])).toBe(existing)
  })

  it('does not mutate the array it was given', () => {
    const existing = [message('1', 'first')]
    upsertMessage(existing, message('2', 'second'))

    expect(existing).toHaveLength(1)
  })

  it('deduplicates the same message id arriving twice', () => {
    const first = upsertMessage([], message('7', 'hello'))
    const second = upsertMessage(first, message('7', 'hello'))

    expect(second).toHaveLength(1)
  })
})
