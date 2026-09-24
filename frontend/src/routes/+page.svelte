<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import ChatArea from '$lib/components/ChatArea.svelte';
	import { api } from '$lib/api';
	import type { Room, Message, RoomParticipant, User, ParticipantDisplay } from '$lib/types';
	import { colorVar } from '$lib/utils';
	import { toast } from '@hermitk/bluenite';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	let user = $state<User | null>(null);
	let rooms = $state<Room[]>([]);
	let allMessages = $state<Message[]>([]);
	let participants = $state<RoomParticipant[]>([]);
	let selectedRoom = $state<Room | null>(null);
	let loading = $state(true);

	const currentUserId = $derived(user?.id ?? '');

	function getParticipant(userId: string): ParticipantDisplay | undefined {
		const p = participants.find((p) => p.id === userId);
		if (!p) return undefined;
		return { name: p.username, color: colorVar(p.color || '--primary') };
	}

	const memberCount = $derived(selectedRoom ? participants.length : 0);

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
		loading = false;
	});

	async function handleSelect(room: Room) {
		selectedRoom = room;
		const [messagesRes, participantsRes] = await Promise.all([
			api.getMessages(room.id),
			api.getRoomParticipants(room.id),
		]);
		if (messagesRes.error) {
			toast.show({ variant: 'danger', message: messagesRes.error });
		}
		if (participantsRes.error) {
			toast.show({ variant: 'danger', message: participantsRes.error });
		}
		allMessages = messagesRes.data || [];
		participants = participantsRes.data || [];
	}

	async function handleSend(content: string) {
		if (!selectedRoom) return;
		const res = await api.sendMessage(selectedRoom.id, content);
		if (res.error) {
			toast.show({ variant: 'danger', message: res.error });
			return;
		}
		if (res.data) allMessages = [...allMessages, res.data];
	}

	async function handleEdit(messageId: string, newContent: string) {
		const res = await api.editMessage(messageId, newContent);
		if (res.error) {
			toast.show({ variant: 'danger', message: res.error });
			return;
		}
		if (res.data) {
			allMessages = allMessages.map((m) => (m.id === messageId ? res.data! : m));
		}
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
		<Sidebar {rooms} selectedRoomId={selectedRoom?.id ?? null} onSelect={handleSelect} />
		<ChatArea
			room={selectedRoom}
			messages={allMessages}
			{currentUserId}
			{memberCount}
			{getParticipant}
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
