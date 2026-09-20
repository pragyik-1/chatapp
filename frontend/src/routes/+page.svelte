<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import ChatArea from '$lib/components/ChatArea.svelte';
	import {
		fakeRooms,
		fakeRoomParticipants,
		fakeMessages,
		fakeUsers,
		fakeCurrentUser,
	} from '$lib/types';
	import type { Room, Message } from '$lib/types';

	// ---- Fake preview data ----
	// Swap these out for real backend calls via `api` from '$lib/api' when wiring up.
	let rooms = $state<Room[]>(fakeRooms);
	let allMessages = $state<Message[]>(fakeMessages);

	const currentUserId = fakeCurrentUser.id;
	const getParticipant = (userId: string) => fakeUsers[userId];
	const memberCount = (roomId: string) => fakeRoomParticipants[roomId]?.length ?? 0;

	let selectedRoom = $state<Room | null>(null);

	let filteredMessages = $derived(
		selectedRoom ? allMessages.filter((m) => m.room_id === selectedRoom!.id) : []
	);

	function handleSelect(room: Room) {
		selectedRoom = room;
	}

	function handleSend(content: string) {
		if (!selectedRoom) return;
		const newMsg: Message = {
			id: `msg-${Date.now()}`,
			room_id: selectedRoom.id,
			sender_id: currentUserId,
			content,
			reply_to_id: null,
			is_edited: false,
			edited_at: null,
			created_at: new Date().toISOString(),
		};
		allMessages = [...allMessages, newMsg];
	}

	function handleEdit(messageId: string, newContent: string) {
		allMessages = allMessages.map((m) =>
			m.id === messageId
				? { ...m, content: newContent, is_edited: true, edited_at: new Date().toISOString() }
				: m
		);
	}

	function handleDelete(messageId: string) {
		allMessages = allMessages.filter((m) => m.id !== messageId);
	}
</script>

<div class="app-shell">
	<Sidebar {rooms} selectedRoomId={selectedRoom?.id ?? null} onSelect={handleSelect} />
	<ChatArea
		room={selectedRoom}
		messages={filteredMessages}
		currentUserId={currentUserId}
		memberCount={selectedRoom ? memberCount(selectedRoom.id) : 0}
		{getParticipant}
		onSend={handleSend}
		onEdit={handleEdit}
		onDelete={handleDelete}
	/>
</div>

<style>
	.app-shell {
		display: flex;
		height: calc(100vh - 52px);
		overflow: hidden;
	}
</style>
