export type User = {
  id: string
  username: string
  email: string
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
}

export type RefreshTokenRequest = {
  refresh_token: string
}

export type CreateRoomRequest = {
  name: string
  is_group: boolean
}

export type RoomParticipantRequest = {
  user_id: string
}

export type SendMessageRequest = {
  room_id: string
  content: string
  reply_to_id?: string | null
}

export type EditMessageRequest = {
  content: string
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
  language?: string
  notifications_enabled?: boolean
}

export type ParticipantDisplay = {
  name: string
  color: string
}

export type UserSearchResult = {
  id: string
  username: string
  email: string
  status: number | null
  last_seen: string | null
  created_at: string
  color: string | null
}
