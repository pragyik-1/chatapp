import type { Message, Room, User } from "$lib/types";
import { getCookie } from "$lib/utils";

export class Api {
    private baseUrl: string;
    private token: string | null = null;

    constructor(baseUrl: string) {
        this.baseUrl = baseUrl;
        this.token = getCookie("token") || null;
    }

    async registerUser(user: User): Promise<void> {
        const response = await fetch(`${this.baseUrl}/register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(user),
        });
        if (!response.ok) {
            throw new Error(`Failed to register user: ${response.statusText}`);
        }
    }

    async loginUser(email: string, password: string): Promise<void> {
        const response = await fetch(`${this.baseUrl}/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password }),
        });
        if (!response.ok) {
            throw new Error(`Failed to login user: ${response.statusText}`);
        }
        const data = await response.json();
        this.token = data.token;
        document.cookie = `token=${this.token}; path=/; Secure; SameSite=Strict`;
    }

    async refreshToken(): Promise<void> {
        const response = await fetch(`${this.baseUrl}/token/refresh`, {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                ...(this.token ? { 'Authorization': `Bearer ${this.token}` } : {})
            },
        });
        if (!response.ok) {
            throw new Error(`Failed to refresh token: ${response.statusText}`);
        }
        const data = await response.json();
        this.token = data.token;
        document.cookie = `token=${this.token}; path=/; Secure; SameSite=Strict`;
    }

    async getCurrentUser(): Promise<User> {
        const response = await fetch(`${this.baseUrl}/users/me`, {
            headers: { 'Authorization': `Bearer ${this.token}` },
        });
        if (!response.ok) {
            throw new Error(`Failed to fetch current user: ${response.statusText}`);
        }
        return response.json();
    }

    async getUserRooms(userId: string): Promise<Room[]> {
        const response = await fetch(`${this.baseUrl}/users/${userId}/rooms`, {
            headers: { 'Authorization': `Bearer ${this.token}` },
        });
        if (!response.ok) {
            throw new Error(`Failed to fetch rooms for user ${userId}: ${response.statusText}`);
        }
        return response.json();
    }

    async createRoom(roomData: Omit<Room, 'id' | 'createdAt'>): Promise<Room> {
        const response = await fetch(`${this.baseUrl}/rooms/create`, {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.token}` 
            },
            body: JSON.stringify(roomData),
        });
        if (!response.ok) {
            throw new Error(`Failed to create room: ${response.statusText}`);
        }
        return response.json();
    }

    async getRoomParticipants(roomId: string): Promise<User[]> {
        const response = await fetch(`${this.baseUrl}/rooms/${roomId}/participants`, {
            headers: { 'Authorization': `Bearer ${this.token}` },
        });
        if (!response.ok) {
            throw new Error(`Failed to fetch participants for room ${roomId}: ${response.statusText}`);
        }
        return response.json();
    }

    async addParticipant(roomId: string, userId: string): Promise<void> {
        const response = await fetch(`${this.baseUrl}/rooms/${roomId}/participants`, {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.token}` 
            },
            body: JSON.stringify({ userId }),
        });
        if (!response.ok) {
            throw new Error(`Failed to add participant to room ${roomId}: ${response.statusText}`);
        }
    }

    async removeParticipant(roomId: string, userId: string): Promise<void> {
        const response = await fetch(`${this.baseUrl}/rooms/${roomId}/participants`, {
            method: 'DELETE',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.token}` 
            },
            body: JSON.stringify({ userId }),
        });
        if (!response.ok) {
            throw new Error(`Failed to remove participant from room ${roomId}: ${response.statusText}`);
        }
    }

    async getMessages(roomId: string): Promise<Message[]> {
        const response = await fetch(`${this.baseUrl}/rooms/${roomId}/messages`, {
            headers: { 'Authorization': `Bearer ${this.token}` },
        });
        if (!response.ok) {
            throw new Error(`Failed to fetch messages for room ${roomId}: ${response.statusText}`);
        }
        return response.json();
    }

    async sendMessage(roomId: string, content: string): Promise<Message> {
        const response = await fetch(`${this.baseUrl}/rooms/${roomId}/messages`, {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.token}` 
            },
            body: JSON.stringify({ content }),
        });
        if (!response.ok) {
            throw new Error(`Failed to send message to room ${roomId}: ${response.statusText}`);
        }
        return response.json();
    }

    async editMessage(messageId: string, content: string): Promise<Message> {
        const response = await fetch(`${this.baseUrl}/rooms/messages/${messageId}`, {
            method: 'PUT',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.token}` 
            },
            body: JSON.stringify({ content }),
        });
        if (!response.ok) {
            throw new Error(`Failed to edit message ${messageId}: ${response.statusText}`);
        }
        return response.json();
    }

    async deleteMessage(messageId: string): Promise<void> {
        const response = await fetch(`${this.baseUrl}/rooms/messages/${messageId}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${this.token}` },
        });
        if (!response.ok) {
            throw new Error(`Failed to delete message ${messageId}: ${response.statusText}`);
        }
    }
}

export const api = new Api('http://localhost:8080');