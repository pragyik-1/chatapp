import type { Message, Room, User } from "$lib/types";

export class Api {
    private baseUrl: string;
    private token: string | null = null;

    constructor(baseUrl: string) {
        this.baseUrl = baseUrl;
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
    }

    async getRooms(): Promise<Room[]> {
        const response = await fetch(`${this.baseUrl}/rooms`);
        if (!response.ok) {
            throw new Error(`Failed to fetch rooms: ${response.statusText}`);
        }
        return response.json();
    }

    async getMessages(roomId: string): Promise<Message[]> {
        const response = await fetch(`${this.baseUrl}/rooms/${roomId}/messages`);
        if (!response.ok) {
            throw new Error(`Failed to fetch messages for room ${roomId}: ${response.statusText}`);
        }
        return response.json();
    }

    async sendMessage(roomId: string, content: string): Promise<Message> {
        const response = await fetch(`${this.baseUrl}/rooms/${roomId}/messages`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', ...(this.token ? { 'Authorization': `Bearer ${this.token}` } : {}) },
            body: JSON.stringify({ content }),
        });
        if (!response.ok) {
            throw new Error(`Failed to send message to room ${roomId}: ${response.statusText}`);
        }
        return response.json();
    }

    async editMessage(messageId: string, content: string): Promise<Message> {
        const response = await fetch(`${this.baseUrl}/messages/${messageId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json', ...(this.token ? { 'Authorization': `Bearer ${this.token}` } : {}) },
            body: JSON.stringify({ content }),
        });
        if (!response.ok) {
            throw new Error(`Failed to edit message ${messageId}: ${response.statusText}`);
        }
        return response.json();
    }

    async deleteMessage(messageId: string): Promise<void> {
        const response = await fetch(`${this.baseUrl}/messages/${messageId}`, {
            method: 'DELETE',
            headers: { ...(this.token ? { 'Authorization': `Bearer ${this.token}` } : {}) },
        });
        if (!response.ok) {
            throw new Error(`Failed to delete message ${messageId}: ${response.statusText}`);
        }
    }
}