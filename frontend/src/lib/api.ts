import type {
  User,
  Room,
  RoomParticipant,
  Message,
  UserSettings,
  UpdateUserSettingsRequest,
  RegisterUserRequest,
  LoginUserRequest,
  CreateRoomRequest,
  SendMessageRequest,
  ApiResponse,
  LoginUserResponse,
  UserSearchResult,
} from '$lib/types'
import { getCookie, setCookie, deleteCookie, isTokenValid } from '$lib/utils'

export const API_BASE_URL = 'http://localhost:8080'

const ACCESS_TOKEN_TTL_DAYS = 1

export class Api {
  private baseUrl: string
  private accessToken: string | null = null
  /** Debounced refresh call so concurrent authed requests share one renewal. */
  private refreshInFlight: Promise<string | null> | null = null

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl
    this.accessToken = getCookie('access_token') || null
  }

  resetAuth(): void {
    this.accessToken = null
    deleteCookie('access_token')
  }

  /**
   * Validates the current access token and, when it is missing or expired,
   * automatically renews it via the refresh token. The renewed JWT is cached
   * in the `access_token` cookie and returned. Returns null when no valid
   * session exists.
   */
  async getValidAuthToken(): Promise<string | null> {
    const current = getCookie('access_token') || this.accessToken
    if (current && isTokenValid(current)) return current

    if (!this.refreshInFlight) {
      this.refreshInFlight = this.renewAccessToken().finally(() => {
        this.refreshInFlight = null
      })
    }
    return this.refreshInFlight
  }

  private async renewAccessToken(): Promise<string | null> {
    const { data } = await this.refreshToken()
    return data?.access_token ?? null
  }

  async loginUser(user: LoginUserRequest): Promise<ApiResponse<LoginUserResponse>> {
    const { data, error } = await this.request<LoginUserResponse>(`${this.baseUrl}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(user),
    })
    if (data) {
      this.accessToken = data.access_token
      setCookie('access_token', this.accessToken, ACCESS_TOKEN_TTL_DAYS)
    }
    return { data, error }
  }

  async logoutUser(): Promise<ApiResponse<null>> {
    const token = await this.getValidAuthToken()
    if (!token) {
      // No valid session to revoke server-side; clear local credentials anyway.
      this.resetAuth()
      return { data: null, error: null }
    }
    const { data, error } = await this.request<null>(`${this.baseUrl}/users/me/logout`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })
    if (!error) {
      this.resetAuth()
    }
    return { data, error }
  }

  /**
   * Renews the access token. Pass `refreshToken` when the raw token is only
   * available to the caller (e.g. the SvelteKit hook reads the HttpOnly cookie
   * server-side); when omitted, the backend reads the cookie directly.
   */
  async refreshToken(refreshToken?: string): Promise<ApiResponse<{ access_token: string }>> {
    const options: RequestInit = { method: 'POST' }
    if (refreshToken) {
      options.headers = { 'Content-Type': 'application/json' }
      options.body = JSON.stringify({ refresh_token: refreshToken })
    }
    const { data, error } = await this.request<{ access_token: string }>(
      `${this.baseUrl}/token/refresh`,
      options,
    )
    if (data) {
      setCookie('access_token', data.access_token, ACCESS_TOKEN_TTL_DAYS)
      this.accessToken = data.access_token
    }
    return { data, error }
  }

  async registerUser(user: RegisterUserRequest): Promise<ApiResponse<User>> {
    return this.request<User>(`${this.baseUrl}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(user),
    })
  }

  async getCurrentUser(): Promise<ApiResponse<User>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<User>(`${this.baseUrl}/users/me`, {
      headers: { Authorization: `Bearer ${token}` },
    })
  }

  async getUserSettings(): Promise<ApiResponse<UserSettings>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<UserSettings>(`${this.baseUrl}/users/me/settings`, {
      headers: { Authorization: `Bearer ${token}` },
    })
  }

  async updateUserSettings(patch: UpdateUserSettingsRequest): Promise<ApiResponse<UserSettings>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<UserSettings>(`${this.baseUrl}/users/me/settings`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(patch),
    })
  }

  async getUserRooms(): Promise<ApiResponse<Room[]>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<Room[]>(`${this.baseUrl}/users/me/rooms`, {
      headers: { Authorization: `Bearer ${token}` },
    })
  }

  async searchUsers(query: string): Promise<ApiResponse<UserSearchResult[]>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    const params = new URLSearchParams({ q: query })
    return this.request<UserSearchResult[]>(`${this.baseUrl}/users/search?${params}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
  }

  async createRoom(roomData: CreateRoomRequest): Promise<ApiResponse<Room>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<Room>(`${this.baseUrl}/rooms/create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(roomData),
    })
  }

  async getRoomParticipants(roomId: string): Promise<ApiResponse<RoomParticipant[]>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<RoomParticipant[]>(`${this.baseUrl}/rooms/${roomId}/participants`, {
      headers: { Authorization: `Bearer ${token}` },
    })
  }

  async addParticipant(roomId: string, userId: string): Promise<ApiResponse<{ message: string }>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<{ message: string }>(`${this.baseUrl}/rooms/${roomId}/participants`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ user_id: userId }),
    })
  }

  async removeParticipant(
    roomId: string,
    userId: string,
  ): Promise<ApiResponse<{ message: string }>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<{ message: string }>(`${this.baseUrl}/rooms/${roomId}/participants`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ user_id: userId }),
    })
  }

  async getMessages(roomId: string): Promise<ApiResponse<Message[]>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<Message[]>(`${this.baseUrl}/rooms/${roomId}/messages`, {
      headers: { Authorization: `Bearer ${token}` },
    })
  }

  async sendMessage(
    roomId: string,
    content: string,
    replyToId?: string | null,
  ): Promise<ApiResponse<Message>> {
    const body: SendMessageRequest = {
      room_id: roomId,
      content,
    }
    if (replyToId !== undefined) {
      body.reply_to_id = replyToId
    }
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<Message>(`${this.baseUrl}/rooms/${roomId}/messages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    })
  }

  async editMessage(messageId: string, content: string): Promise<ApiResponse<Message>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<Message>(`${this.baseUrl}/rooms/messages/${messageId}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ content }),
    })
  }

  async deleteMessage(messageId: string): Promise<ApiResponse<null>> {
    const token = await this.getValidAuthToken()
    if (!token) return { data: null, error: 'Not authenticated' }
    return this.request<null>(`${this.baseUrl}/rooms/messages/${messageId}`, {
      method: 'DELETE',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })
  }

  private async request<T>(url: string, options?: RequestInit): Promise<ApiResponse<T>> {
    try {
      const response = await fetch(url, { ...options, credentials: 'include' })
      if (!response.ok) {
        let errorMessage = response.statusText
        try {
          const body = await response.json()
          if (body.error) errorMessage = body.error
        } catch {
          // response has no JSON body to read the error message from
        }
        return { data: null, error: errorMessage }
      }
      if (response.status === 204) {
        return { data: null as unknown as T, error: null }
      }
      const data = await response.json()
      return { data, error: null }
    } catch (err) {
      return { data: null, error: err instanceof Error ? err.message : 'Network error' }
    }
  }
}

export const api = new Api(API_BASE_URL)
