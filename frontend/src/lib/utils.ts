import { WEBSOCKET_PATH, WEBSOCKET_TOKEN_PARAM } from '$lib/constants'
import type { Message } from '$lib/types'

const sharedCookieOptions = 'path=/; Secure; SameSite=Strict'

export function getCookie(name: string): string | null {
  if (typeof document === 'undefined') return null
  const value = `; ${document.cookie}`
  const parts = value.split(`; ${name}=`)
  if (parts.length === 2) {
    return parts.pop()?.split(';').shift() || null
  }
  return null
}

export function setCookie(name: string, value: string, days: number): void {
  if (typeof document === 'undefined') return
  const expires = new Date(Date.now() + days * 864e5).toUTCString()
  document.cookie = `${name}=${value}; expires=${expires}; ${sharedCookieOptions}`
}

export function deleteCookie(name: string): void {
  if (typeof document === 'undefined') return
  document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; ${sharedCookieOptions}`
}

export function colorVar(color: string): string {
  return `var(${color})`
}

/**
 * Derives a short initials label from a username. Since the backend does not
 * store first/last names, multi-word names use the first letter of each word
 * (up to two words); single-word names fall back to their first two letters.
 */
export function getInitials(name: string): string {
  const trimmed = name.trim()
  if (!trimmed) return '?'
  const parts = trimmed.split(/\s+/)
  if (parts.length > 1) {
    return parts
      .slice(0, 2)
      .map((part) => part.charAt(0))
      .join('')
      .toUpperCase()
  }
  return trimmed.slice(0, 2).toUpperCase()
}

const TOKEN_EXPIRY_MARGIN_MS = 15_000

function decodeJwtPayload(token: string): Record<string, unknown> | null {
  const part = token.split('.')[1]
  if (!part) return null
  try {
    const base64 = part.replace(/-/g, '+').replace(/_/g, '/')
    const decoded = decodeURIComponent(
      atob(base64)
        .split('')
        .map((c) => '%' + c.charCodeAt(0).toString(16).padStart(2, '0'))
        .join(''),
    )
    return JSON.parse(decoded) as Record<string, unknown>
  } catch {
    return null
  }
}

/**
 * Derives a WebSocket URL from the HTTP base URL, carrying an access token as a
 * query parameter. Browsers cannot set an Authorization header on a WebSocket
 * handshake, so the token travels in the query string instead.
 *
 * The scheme is mapped rather than hard-coded so a secure origin yields `wss:`
 * without a second URL constant.
 */
export function websocketUrl(baseUrl: string, token: string): string {
  const url = new URL(baseUrl)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  url.pathname = WEBSOCKET_PATH
  url.search = new URLSearchParams({ [WEBSOCKET_TOKEN_PARAM]: token }).toString()
  return url.toString()
}

/**
 * Inserts or replaces a message in a list keyed by id, preserving order.
 *
 * Used for both a newly received message and an edited one. Deduplicating by id
 * is what lets a message the local user just sent be appended by the REST
 * response and then be ignored when its realtime event arrives.
 */
export function upsertMessage(messages: Message[], message: Message): Message[] {
  const index = messages.findIndex((m) => m.id === message.id)
  if (index === -1) return [...messages, message]
  if (messages[index] === message) return messages
  const next = [...messages]
  next[index] = message
  return next
}

/**
 * Client-side check that an access token is structurally sound and not close to
 * expiring. The signature cannot be verified without the server secret, so an
 * expired or malformed token simply triggers a refresh via the refresh token.
 */
export function isTokenValid(token: string | null): boolean {
  if (!token) return false
  const payload = decodeJwtPayload(token)
  if (!payload || typeof payload.exp !== 'number') return false
  return payload.exp * 1000 > Date.now() + TOKEN_EXPIRY_MARGIN_MS
}
