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

export type MessageDeletedPayload = {
  message_id: string
  room_id: string
}

export type RealtimeStatus = 'connecting' | 'open' | 'reconnecting' | 'offline'

/**renaming one is a breaking change for every deployed client.*/
export type ServerEventType = 'message_created' | 'message_updated' | 'message_deleted'

/** A frame pushed by the server. `data` matches the listed event's payload. */
export type ServerEvent = {
  type: ServerEventType
  data: Message | MessageDeletedPayload
}

export type ClientEventType = 'subscribe' | 'unsubscribe'

export type ClientEvent = {
  type: ClientEventType
  room_id?: string
}
