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
} from '$lib/types'
import { getCookie, setCookie, deleteCookie } from '$lib/utils'

export class Api {
  private baseUrl: string
  private accessToken: string | null = null

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl
    this.accessToken = getCookie('access_token') || null
  }

  private get authToken(): string | null {
    return getCookie('access_token') || this.accessToken
  }

  resetAuth(): void {
    this.accessToken = null
    deleteCookie('access_token')
  }

  async loginUser(user: LoginUserRequest): Promise<ApiResponse<LoginUserResponse>> {
    const { data, error } = await this.request<LoginUserResponse>(`${this.baseUrl}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(user),
    })
    if (data) {
      this.accessToken = data.access_token
      setCookie('access_token', this.accessToken, 1)
    }
    return { data, error }
  }

  async logoutUser(): Promise<ApiResponse<null>> {
    const { data, error } = await this.request<null>(`${this.baseUrl}/users/me/logout`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${this.authToken}`,
      },
    })
    if (!error) {
      this.resetAuth()
    }
    return { data, error }
  }

  async refreshToken(refreshToken: string): Promise<ApiResponse<{ access_token: string }>> {
    const { data, error } = await this.request<{ access_token: string }>(
      `${this.baseUrl}/token/refresh`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      },
    )
    if (data) {
      setCookie('access_token', data.access_token, 1)
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
    return this.request<User>(`${this.baseUrl}/users/me`, {
      headers: { Authorization: `Bearer ${this.authToken}` },
    })
  }

  async getUserSettings(): Promise<ApiResponse<UserSettings>> {
    return this.request<UserSettings>(`${this.baseUrl}/users/me/settings`, {
      headers: { Authorization: `Bearer ${this.authToken}` },
    })
  }

  async updateUserSettings(patch: UpdateUserSettingsRequest): Promise<ApiResponse<UserSettings>> {
    return this.request<UserSettings>(`${this.baseUrl}/users/me/settings`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.authToken}`,
      },
      body: JSON.stringify(patch),
    })
  }

  async getUserRooms(): Promise<ApiResponse<Room[]>> {
    return this.request<Room[]>(`${this.baseUrl}/users/me/rooms`, {
      headers: { Authorization: `Bearer ${this.authToken}` },
    })
  }

  async createRoom(roomData: CreateRoomRequest): Promise<ApiResponse<Room>> {
    return this.request<Room>(`${this.baseUrl}/rooms/create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.authToken}`,
      },
      body: JSON.stringify(roomData),
    })
  }

  async getRoomParticipants(roomId: string): Promise<ApiResponse<RoomParticipant[]>> {
    return this.request<RoomParticipant[]>(`${this.baseUrl}/rooms/${roomId}/participants`, {
      headers: { Authorization: `Bearer ${this.authToken}` },
    })
  }

  async addParticipant(roomId: string, userId: string): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>(`${this.baseUrl}/rooms/${roomId}/participants`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.authToken}`,
      },
      body: JSON.stringify({ user_id: userId }),
    })
  }

  async removeParticipant(
    roomId: string,
    userId: string,
  ): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>(`${this.baseUrl}/rooms/${roomId}/participants`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.authToken}`,
      },
      body: JSON.stringify({ user_id: userId }),
    })
  }

  async getMessages(roomId: string): Promise<ApiResponse<Message[]>> {
    return this.request<Message[]>(`${this.baseUrl}/rooms/${roomId}/messages`, {
      headers: { Authorization: `Bearer ${this.authToken}` },
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
    return this.request<Message>(`${this.baseUrl}/rooms/${roomId}/messages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.authToken}`,
      },
      body: JSON.stringify(body),
    })
  }

  async editMessage(messageId: string, content: string): Promise<ApiResponse<Message>> {
    return this.request<Message>(`${this.baseUrl}/rooms/messages/${messageId}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.authToken}`,
      },
      body: JSON.stringify({ content }),
    })
  }

  async deleteMessage(messageId: string): Promise<ApiResponse<null>> {
    return this.request<null>(`${this.baseUrl}/rooms/messages/${messageId}`, {
      method: 'DELETE',
      headers: {
        Authorization: `Bearer ${this.authToken}`,
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

export const api = new Api('http://localhost:8080')
