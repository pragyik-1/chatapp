import { colorVar } from '$lib/utils'

export type User = {
  id: string
  username: string
  email: string
  password_hash: string
  status: number | null
  last_seen: string | null
  created_at: string
}

export type Room = {
  id: string
  name: string | null
  is_group: boolean
  created_by: string
  created_at: string
}

export type Message = {
  id: string
  room_id: string
  sender_id: string
  content: string
  reply_to_id: string | null
  is_edited: boolean
  edited_at: string | null
  created_at: string
}

export type RoomParticipant = {
  id: string
  username: string
  email: string
  status: number | null
  last_seen: string | null
  color: string | null
  joined_at: string
}

export type RegisterUserRequest = {
  username: string
  email: string
  password: string
  color?: string
}

export type LoginUserRequest = {
  email: string
  password: string
}

export type LoginUserResponse = {
  access_token: string
  refresh_token_hash: string
}

export type RefreshTokenRequest = {
  refresh_token_hash: string
}

export type CreateRoomRequest = {
  name: string
  is_group: boolean
  created_by: string
}

export type RoomParticipantRequest = {
  user_id: string
}

export type SendMessageRequest = {
  room_id: string
  sender_id: string
  content: string
  reply_to_id?: string | null
}

export type EditMessageRequest = {
  sender_id: string
  content: string
}

export type DeleteMessageRequest = {
  sender_id: string
}

export type ApiResponse<T> = {
  data: T | null
  error: string | null
}

export type UserSettings = {
  user_id: string
  color: string
  language: string
  notifications_enabled: boolean
  updated_at: string
}

export type UpdateUserSettingsRequest = {
  color?: string
  theme?: string
  language?: string
  notifications_enabled?: boolean
}

// ---- Fake data for previewing the UI ----
// Replace these with real backend responses when wiring up the API.

export type ParticipantDisplay = {
  name: string
  color: string
}

export const fakeUsers: Record<string, ParticipantDisplay> = {
  'user-1': { name: 'You', color: colorVar('--primary') },
  'user-2': { name: 'Alice', color: colorVar('--success') },
  'user-3': { name: 'Bob', color: colorVar('--warn') },
  'user-4': { name: 'Charlie', color: colorVar('--secondary') },
}

export const fakeCurrentUser: User = {
  id: 'user-1',
  username: 'You',
  email: 'you@example.com',
  password_hash: 'fake',
  status: 1,
  last_seen: '2026-09-07T10:00:00Z',
  created_at: '2026-01-01T00:00:00Z',
}

export const fakeRooms: Room[] = [
  {
    id: 'room-1',
    name: 'Alice',
    is_group: false,
    created_by: 'user-1',
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'room-2',
    name: 'Bob',
    is_group: false,
    created_by: 'user-1',
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'room-3',
    name: 'General',
    is_group: true,
    created_by: 'user-1',
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'room-4',
    name: 'Design Team',
    is_group: true,
    created_by: 'user-1',
    created_at: '2026-01-01T00:00:00Z',
  },
]

export const fakeRoomParticipants: Record<string, string[]> = {
  'room-1': ['user-1', 'user-2'],
  'room-2': ['user-1', 'user-3'],
  'room-3': ['user-1', 'user-2', 'user-3', 'user-4'],
  'room-4': ['user-1', 'user-2', 'user-4'],
}

export const fakeMessages: Message[] = [
  {
    id: 'msg-1',
    room_id: 'room-1',
    sender_id: 'user-2',
    content: 'Hey! How are you?',
    reply_to_id: null,
    is_edited: false,
    edited_at: null,
    created_at: '2026-09-07T09:15:00Z',
  },
  {
    id: 'msg-2',
    room_id: 'room-1',
    sender_id: 'user-1',
    content: 'Doing well, thanks! Just finished the layout work.',
    reply_to_id: null,
    is_edited: false,
    edited_at: null,
    created_at: '2026-09-07T09:16:00Z',
  },
  {
    id: 'msg-3',
    room_id: 'room-1',
    sender_id: 'user-2',
    content: 'Nice! See you tomorrow!',
    reply_to_id: null,
    is_edited: false,
    edited_at: null,
    created_at: '2026-09-07T09:30:00Z',
  },
  {
    id: 'msg-4',
    room_id: 'room-2',
    sender_id: 'user-3',
    content: 'Can you review my PR?',
    reply_to_id: null,
    is_edited: false,
    edited_at: null,
    created_at: '2026-09-07T11:02:00Z',
  },
  {
    id: 'msg-5',
    room_id: 'room-2',
    sender_id: 'user-1',
    content: 'Sure, I will take a look.',
    reply_to_id: null,
    is_edited: false,
    edited_at: null,
    created_at: '2026-09-07T11:10:00Z',
  },
  {
    id: 'msg-6',
    room_id: 'room-3',
    sender_id: 'user-4',
    content: 'Good morning everyone! Standup in 10 minutes.',
    reply_to_id: null,
    is_edited: false,
    edited_at: null,
    created_at: '2026-09-07T13:00:00Z',
  },
  {
    id: 'msg-7',
    room_id: 'room-3',
    sender_id: 'user-2',
    content: 'Morning! I’ll be there.',
    reply_to_id: null,
    is_edited: false,
    edited_at: null,
    created_at: '2026-09-07T13:01:00Z',
  },
  {
    id: 'msg-8',
    room_id: 'room-3',
    sender_id: 'user-3',
    content: 'Anyone up for lunch?',
    reply_to_id: null,
    is_edited: false,
    edited_at: null,
    created_at: '2026-09-07T13:05:00Z',
  },
  {
    id: 'msg-9',
    room_id: 'room-4',
    sender_id: 'user-4',
    content: 'Check out the new mockups I pushed.',
    reply_to_id: null,
    is_edited: false,
    edited_at: null,
    created_at: '2026-09-06T16:20:00Z',
  },
  {
    id: 'msg-10',
    room_id: 'room-4',
    sender_id: 'user-2',
    content: 'Looks great! The colors really pop.',
    reply_to_id: null,
    is_edited: true,
    edited_at: '2026-09-06T16:25:00Z',
    created_at: '2026-09-06T16:24:00Z',
  },
]
