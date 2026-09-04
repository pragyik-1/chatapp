export type User = {
  id: string
  name: string
  color: string
}

export type RoomType = 'dm' | 'group'

export type Room = {
  id: string
  name: string
  type: RoomType
  participants: string[]
  lastMessage?: string
  lastMessageTime?: string
}

export type Message = {
  id: string
  roomId: string
  userId: string
  content: string
  timestamp: string
  edited?: boolean
}

export const currentUser: User = {
  id: 'user-1',
  name: 'You',
  color: 'var(--primary)',
}

export const users: User[] = [
  currentUser,
  { id: 'user-2', name: 'Alice', color: 'var(--success)' },
  { id: 'user-3', name: 'Bob', color: 'var(--warn)' },
  { id: 'user-4', name: 'Charlie', color: 'var(--secondary)' },
]

export const rooms: Room[] = [
  {
    id: 'room-1',
    name: 'Alice',
    type: 'dm',
    participants: ['user-1', 'user-2'],
    lastMessage: 'See you tomorrow!',
    lastMessageTime: '10:30 AM',
  },
  {
    id: 'room-2',
    name: 'Bob',
    type: 'dm',
    participants: ['user-1', 'user-3'],
    lastMessage: 'Thanks for the help',
    lastMessageTime: '9:15 AM',
  },
  {
    id: 'room-3',
    name: 'General',
    type: 'group',
    participants: ['user-1', 'user-2', 'user-3', 'user-4'],
    lastMessage: 'Anyone up for lunch?',
    lastMessageTime: '11:00 AM',
  },
  {
    id: 'room-4',
    name: 'Design Team',
    type: 'group',
    participants: ['user-1', 'user-2', 'user-4'],
    lastMessage: 'Check out the new mockups',
    lastMessageTime: 'Yesterday',
  },
]

export const messages: Message[] = [
  {
    id: 'msg-1',
    roomId: 'room-1',
    userId: 'user-2',
    content: 'Hey! How are you?',
    timestamp: '10:25 AM',
  },
  {
    id: 'msg-2',
    roomId: 'room-1',
    userId: 'user-1',
    content: 'Doing well, thanks!',
    timestamp: '10:26 AM',
  },
  {
    id: 'msg-3',
    roomId: 'room-1',
    userId: 'user-2',
    content: 'See you tomorrow!',
    timestamp: '10:30 AM',
  },
  {
    id: 'msg-4',
    roomId: 'room-2',
    userId: 'user-3',
    content: 'Can you review my PR?',
    timestamp: '9:00 AM',
  },
  {
    id: 'msg-5',
    roomId: 'room-2',
    userId: 'user-1',
    content: 'Sure, I will take a look',
    timestamp: '9:10 AM',
  },
  {
    id: 'msg-6',
    roomId: 'room-2',
    userId: 'user-3',
    content: 'Thanks for the help',
    timestamp: '9:15 AM',
  },
  {
    id: 'msg-7',
    roomId: 'room-3',
    userId: 'user-4',
    content: 'Good morning everyone!',
    timestamp: '10:45 AM',
  },
  { id: 'msg-8', roomId: 'room-3', userId: 'user-2', content: 'Morning!', timestamp: '10:46 AM' },
  {
    id: 'msg-9',
    roomId: 'room-3',
    userId: 'user-3',
    content: 'Anyone up for lunch?',
    timestamp: '11:00 AM',
  },
  {
    id: 'msg-10',
    roomId: 'room-4',
    userId: 'user-4',
    content: 'Check out the new mockups',
    timestamp: 'Yesterday',
  },
  {
    id: 'msg-11',
    roomId: 'room-4',
    userId: 'user-2',
    content: 'Looks great!',
    timestamp: 'Yesterday',
  },
]
