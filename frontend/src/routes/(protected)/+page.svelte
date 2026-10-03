<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import ChatArea from '$lib/components/ChatArea.svelte';
	import { api } from '$lib/api';
	import type {
		Room,
		Message,
		MessageDeletedPayload,
		RealtimeStatus,
		RoomParticipant,
		User,
		ParticipantDisplay,
		UserSearchResult,
	} from '$lib/types';
	import { colorVar, upsertMessage } from '$lib/utils';
	import { realtime } from '$lib/realtime';
	import { toast } from '@hermitk/bluenite';
	import { onDestroy, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	let user = $state<User | null>(null);
	let rooms = $state<Room[]>([]);
	let allMessages = $state<Message[]>([]);
	let participants = $state<RoomParticipant[]>([]);
	let selectedRoom = $state<Room | null>(null);
	let loading = $state(true);
	let connectionState = $state<RealtimeStatus>(realtime.getStatus());

	/** The room the socket is currently subscribed to, so it can be left. */
	let subscribedRoomId = $state<string | null>(null);

	let pendingDM = $state<UserSearchResult | null>(null);
	let dmNames = $state<Record<string, string>>({});
	let dmOwners = $state<Record<string, string>>({});
	let dmColors = $state<Record<string, string>>({});

	const currentUserId = $derived(user?.id ?? '');

	const selectedRoomName = $derived(
		selectedRoom ? (dmNames[selectedRoom.id] ?? selectedRoom.name ?? 'Direct Message') : ''
	);

	function getParticipant(userId: string): ParticipantDisplay | undefined {
		const p = participants.find((p) => p.id === userId);
		if (!p) return undefined;
		return { name: p.username, color: colorVar(p.color || '--primary') };
	}

	const memberCount = $derived(selectedRoom ? participants.length : 0);

	async function enrichDmRooms(roomList: Room[]) {
		const dmRooms = roomList.filter((r) => !r.is_group && !r.name);
		await Promise.all(
			dmRooms.map(async (room) => {
				const res = await api.getRoomParticipants(room.id);
				if (res.error || !res.data) {
					dmNames[room.id] = 'Direct Message';
					return;
				}
				const other = res.data.find((p) => p.id !== currentUserId);
				if (other) {
					dmOwners[room.id] = other.id;
					dmNames[room.id] = other.username;
					dmColors[room.id] = other.color || '--primary';
				} else {
					dmNames[room.id] = 'Direct Message';
				}
			})
		);
	}

	onMount(() => {
		realtime.onStatus((status) => {
			connectionState = status;
		});
		realtime.onMessageCreated(applyIncomingMessage);
		realtime.onMessageUpdated(applyIncomingMessage);
		realtime.onMessageDeleted(applyRemovedMessage);
	});

	onMount(async () => {
		const [userRes, roomsRes] = await Promise.all([api.getCurrentUser(), api.getUserRooms()]);
		if (userRes.error || !userRes.data) {
			toast.show({ variant: 'danger', message: userRes.error || 'Failed to load your profile' });
			goto(resolve('/login'));
			return;
		}
		if (roomsRes.error) {
			toast.show({ variant: 'danger', message: roomsRes.error });
		}
		user = userRes.data;
		rooms = roomsRes.data || [];
		await enrichDmRooms(rooms);
		loading = false;
	});

	// Leaving the page unsubscribes from the room on screen, so the backend stops
	// fanning that room's events to a socket nobody is reading.
	onDestroy(() => {
		if (subscribedRoomId) {
			realtime.unsubscribe(subscribedRoomId);
			subscribedRoomId = null;
		}
	});

	async function handleSelect(room: Room) {
		pendingDM = null;
		selectedRoom = room;
		// Subscribe before fetching, so a message sent between the fetch and the
		// subscription is still delivered rather than lost in the gap. upsert
		// makes the overlap with the fetched list harmless.
		realtime.subscribe(room.id);
		const [messagesRes, participantsRes] = await Promise.all([
			api.getMessages(room.id),
			api.getRoomParticipants(room.id),
		]);
		// The previous room is only left once the new one is loaded, so no room
		// the user is looking at is ever unsubscribed.
		if (subscribedRoomId && subscribedRoomId !== room.id) {
			realtime.unsubscribe(subscribedRoomId);
		}
		subscribedRoomId = room.id;
		if (messagesRes.error) {
			toast.show({ variant: 'danger', message: messagesRes.error });
		}
		if (participantsRes.error) {
			toast.show({ variant: 'danger', message: participantsRes.error });
		}
		allMessages = messagesRes.data || [];
		participants = participantsRes.data || [];
	}

	// Realtime events are routed by room, because a room switch can be in flight
	// while an event for the room being left arrives.
	function applyIncomingMessage(message: Message) {
		if (!selectedRoom || message.room_id !== selectedRoom.id) return;
		allMessages = upsertMessage(allMessages, message);
	}

	function applyRemovedMessage(payload: MessageDeletedPayload) {
		if (!selectedRoom || payload.room_id !== selectedRoom.id) return;
		allMessages = allMessages.filter((m) => m.id !== payload.message_id);
	}

	// Opens an existing 1:1 room with the target user, or opens a pending
	// conversation with them. Nothing is created on the backend until the first
	// message is sent (handleSend).
	async function handleStartDM(target: UserSearchResult) {
		const existing = rooms.find((r) => !r.is_group && !r.name && dmOwners[r.id] === target.id);
		if (existing) {
			handleSelect(existing);
			return;
		}

		pendingDM = target;
		selectedRoom = null;
		allMessages = [];
		participants = [];
		// No room is on screen, so there is nothing to watch. The subscription
		// returns once handleSend creates the room.
		if (subscribedRoomId) {
			realtime.unsubscribe(subscribedRoomId);
			subscribedRoomId = null;
		}
	}

	async function handleSend(content: string) {
		// A pending DM: create the nameless non-group room and add the recipient
		// as a participant before sending the first message.
		if (!selectedRoom && pendingDM) {
			const target = pendingDM;
			const res = await api.createRoom({ name: '', is_group: false });
			if (res.error || !res.data) {
				toast.show({ variant: 'danger', message: res.error || 'Failed to create room' });
				return;
			}

			const room = res.data;
			const addRes = await api.addParticipant(room.id, target.id);
			if (addRes.error) {
				toast.show({ variant: 'danger', message: addRes.error });
			}

			dmOwners[room.id] = target.id;
			dmNames[room.id] = target.username;
			dmColors[room.id] = target.color || '--primary';
			rooms = [room, ...rooms];
			selectedRoom = room;
			pendingDM = null;
			// The room exists now, so the socket can watch it like any other.
			realtime.subscribe(room.id);
			subscribedRoomId = room.id;

			const partsRes = await api.getRoomParticipants(room.id);
			if (partsRes.error) {
				toast.show({ variant: 'danger', message: partsRes.error });
			}
			participants = partsRes.data || [];
		}

		if (!selectedRoom) return;
		const res = await api.sendMessage(selectedRoom.id, content);
		if (res.error) {
			toast.show({ variant: 'danger', message: res.error });
			return;
		}
		// upsert rather than append: the created event for this same message is
		// already in flight, and appending would show it twice.
		if (res.data) allMessages = upsertMessage(allMessages, res.data);
	}

	async function handleEdit(messageId: string, newContent: string) {
		const res = await api.editMessage(messageId, newContent);
		if (res.error) {
			toast.show({ variant: 'danger', message: res.error });
			return;
		}
		if (res.data) allMessages = upsertMessage(allMessages, res.data);
	}

	async function handleDelete(messageId: string) {
		const res = await api.deleteMessage(messageId);
		if (res.error) {
			toast.show({ variant: 'danger', message: res.error });
			return;
		}
		allMessages = allMessages.filter((m) => m.id !== messageId);
	}
</script>

<div class="app-shell">
	{#if loading}
		<div class="app-loading">Loading…</div>
	{:else}
		<Sidebar
			{rooms}
			selectedRoomId={selectedRoom?.id ?? null}
			roomNames={dmNames}
			roomColors={dmColors}
			onSelect={handleSelect}
			onStartDM={handleStartDM}
		/>
		<ChatArea
			room={selectedRoom}
			roomName={selectedRoomName}
			pendingUser={pendingDM}
			messages={allMessages}
			{currentUserId}
			{memberCount}
			{getParticipant}
			{connectionState}
			onSend={handleSend}
			onEdit={handleEdit}
			onDelete={handleDelete}
		/>
	{/if}
</div>

<style>
	.app-shell {
		display: flex;
		height: calc(100vh - 52px);
		overflow: hidden;
	}

	.app-loading {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--secondary-text);
	}
</style>
