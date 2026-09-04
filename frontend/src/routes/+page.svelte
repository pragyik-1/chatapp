<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import ChatArea from '$lib/components/ChatArea.svelte';
	import { rooms, messages, currentUser } from '$lib/types';
	import type { Room, Message } from '$lib/types';

	let selectedRoom = $state<Room | null>(null);
	let allMessages = $state<Message[]>(messages);

	let filteredMessages = $derived(
		selectedRoom ? allMessages.filter((m) => m.roomId === selectedRoom!.id) : []
	);

	function handleSelect(room: Room) {
		selectedRoom = room;
	}

	function handleSend(content: string) {
		if (!selectedRoom) return;
		const newMsg: Message = {
			id: `msg-${Date.now()}`,
			roomId: selectedRoom.id,
			userId: currentUser.id,
			content,
			timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
		};
		allMessages = [...allMessages, newMsg];
	}

	function handleEdit(messageId: string, newContent: string) {
		allMessages = allMessages.map((m) =>
			m.id === messageId ? { ...m, content: newContent, edited: true } : m
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
		{currentUser}
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
