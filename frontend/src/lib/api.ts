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
  EditMessageRequest,
  DeleteMessageRequest,
  ApiResponse,
} from "$lib/types";
import { getCookie, setCookie } from "$lib/utils";

export class Api {
  private baseUrl: string;
  private accessToken: string | null = null;
  private _refreshToken: string | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
    this.accessToken = getCookie("access_token") || null;
    this._refreshToken = getCookie("refresh_token") || null;
  }

  private async request<T>(url: string, options?: RequestInit): Promise<ApiResponse<T>> {
    try {
      const response = await fetch(url, options);
      if (!response.ok) {
        let errorMessage = response.statusText;
        try {
          const body = await response.json();
          if (body.error) errorMessage = body.error;
        } catch { }
        return { data: null, error: errorMessage };
      }
      if (response.status === 204) {
        return { data: null as unknown as T, error: null };
      }
      const data = await response.json();
      return { data, error: null };
    } catch (err) {
      return { data: null, error: err instanceof Error ? err.message : 'Network error' };
    }
  }

  async registerUser(user: RegisterUserRequest): Promise<ApiResponse<User>> {
    return this.request<User>(`${this.baseUrl}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(user),
    });
  }

  async loginUser(user: LoginUserRequest): Promise<ApiResponse<{ access_token: string; refresh_token: string }>> {
    const { data, error } = await this.request<{ access_token: string; refresh_token: string }>(
      `${this.baseUrl}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(user),
    });
    if (data) {
      this.accessToken = data.access_token;
      this._refreshToken = data.refresh_token;
      setCookie("access_token", this.accessToken, 1);
      setCookie("refresh_token", this._refreshToken, 1);
    }
    return { data, error };
  }

  async refreshToken(): Promise<ApiResponse<{ access_token: string }>> {
    const { data, error } = await this.request<{ access_token: string }>(
      `${this.baseUrl}/token/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: this._refreshToken }),
    });
    if (data) {
      this.accessToken = data.access_token;
      document.cookie = `token=${this.accessToken}; path=/; Secure; SameSite=Strict`;
    }
    return { data, error };
  }

  async getCurrentUser(): Promise<ApiResponse<User>> {
    return this.request<User>(`${this.baseUrl}/users/me`, {
      headers: { 'Authorization': `Bearer ${this.accessToken}` },
    });
  }

  async getUserSettings(): Promise<ApiResponse<UserSettings>> {
    return this.request<UserSettings>(`${this.baseUrl}/users/me/settings`, {
      headers: { 'Authorization': `Bearer ${this.accessToken}` },
    });
  }

  async updateUserSettings(patch: UpdateUserSettingsRequest): Promise<ApiResponse<UserSettings>> {
    return this.request<UserSettings>(`${this.baseUrl}/users/me/settings`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.accessToken}`,
      },
      body: JSON.stringify(patch),
    });
  }

  async getUserRooms(userId: string): Promise<ApiResponse<Room[]>> {
    return this.request<Room[]>(`${this.baseUrl}/users/${userId}/rooms`, {
      headers: { 'Authorization': `Bearer ${this.accessToken}` },
    });
  }

  async createRoom(roomData: CreateRoomRequest): Promise<ApiResponse<Room>> {
    return this.request<Room>(`${this.baseUrl}/rooms/create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.accessToken}`
      },
      body: JSON.stringify(roomData),
    });
  }

  async getRoomParticipants(roomId: string): Promise<ApiResponse<RoomParticipant[]>> {
    return this.request<RoomParticipant[]>(`${this.baseUrl}/rooms/${roomId}/participants`, {
      headers: { 'Authorization': `Bearer ${this.accessToken}` },
    });
  }

  async addParticipant(roomId: string, userId: string): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>(`${this.baseUrl}/rooms/${roomId}/participants`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.accessToken}`
      },
      body: JSON.stringify({ user_id: userId }),
    });
  }

  async removeParticipant(roomId: string, userId: string): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>(`${this.baseUrl}/rooms/${roomId}/participants`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.accessToken}`
      },
      body: JSON.stringify({ user_id: userId }),
    });
  }

  async getMessages(roomId: string): Promise<ApiResponse<Message[]>> {
    return this.request<Message[]>(`${this.baseUrl}/rooms/${roomId}/messages`, {
      headers: { 'Authorization': `Bearer ${this.accessToken}` },
    });
  }

  async sendMessage(roomId: string, senderId: string, content: string, replyToId?: string | null): Promise<ApiResponse<Message>> {
    const body: SendMessageRequest = {
      room_id: roomId,
      sender_id: senderId,
      content,
    };
    if (replyToId !== undefined) {
      body.reply_to_id = replyToId;
    }
    return this.request<Message>(`${this.baseUrl}/rooms/${roomId}/messages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.accessToken}`
      },
      body: JSON.stringify(body),
    });
  }

  async editMessage(messageId: string, senderId: string, content: string): Promise<ApiResponse<Message>> {
    const body: EditMessageRequest = {
      sender_id: senderId,
      content,
    };
    return this.request<Message>(`${this.baseUrl}/rooms/messages/${messageId}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.accessToken}`
      },
      body: JSON.stringify(body),
    });
  }

  async deleteMessage(messageId: string, senderId: string): Promise<ApiResponse<null>> {
    const body: DeleteMessageRequest = {
      sender_id: senderId,
    };
    return this.request<null>(`${this.baseUrl}/rooms/messages/${messageId}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.accessToken}`
      },
      body: JSON.stringify(body),
    });
  }
}

export const api = new Api('http://localhost:8080');
