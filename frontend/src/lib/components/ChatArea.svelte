<script lang="ts">
	import { Button, Col, Dropdown, Textarea } from '@hermitk/bluenite';
	import { EllipsisVertical, Pencil, Trash2 } from 'lucide-svelte';
	import { colorVar, getInitials } from '$lib/utils';
	import type {
		Room,
		Message,
		ParticipantDisplay,
		RealtimeStatus,
		UserSearchResult
	} from '$lib/types';

	let {
		room,
		roomName,
		pendingUser,
		messages,
		currentUserId,
		memberCount,
		getParticipant,
		connectionState = 'offline',
		onSend,
		onEdit,
		onDelete,
	}: {
		room: Room | null;
		roomName?: string;
		pendingUser?: UserSearchResult | null;
		messages: Message[];
		currentUserId: string;
		memberCount: number;
		getParticipant: (userId: string) => ParticipantDisplay | undefined;
		/** Realtime connection state, rendered as a banner. Optional so the
		 *  component still works when no socket is in use. */
		connectionState?: RealtimeStatus;
		onSend: (content: string) => void;
		onEdit: (messageId: string, content: string) => void;
		onDelete: (messageId: string) => void;
	} = $props();

	let input = $state('');
	let messagesEl: HTMLDivElement = $state(undefined!);
	let textareaEl: HTMLTextAreaElement = $state(undefined!);

	let menuOpen = $state(false);
	let menuAnchor = $state<HTMLElement | null>(null);
	let menuMsgId = $state('');

	let editingId = $state<string | null>(null);
	let editContent = $state('');

	function autoResize() {
		const el = textareaEl;
		if (!el) return;
		el.style.height = 'auto';
		el.style.height = `${Math.min(el.scrollHeight, 144)}px`;
	}

	function send() {
		const trimmed = input.trim();
		if (!trimmed || (!room && !pendingUser)) return;
		onSend(trimmed);
		input = '';
		if (textareaEl) textareaEl.style.height = 'auto';
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			send();
		}
	}

	function openMenu(e: MouseEvent, msgId: string) {
		if (menuOpen && menuMsgId === msgId) {
			menuOpen = false;
			return;
		}
		menuAnchor = e.currentTarget as HTMLElement;
		menuMsgId = msgId;
		menuOpen = true;
	}

	function startEdit() {
		const msg = messages.find((m) => m.id === menuMsgId);
		if (!msg) return;
		menuOpen = false;
		editingId = msg.id;
		editContent = msg.content;
	}

	function confirmEdit() {
		if (!editingId) return;
		const trimmed = editContent.trim();
		if (!trimmed) return;
		onEdit(editingId, trimmed);
		editingId = null;
		editContent = '';
	}

	function cancelEdit() {
		editingId = null;
		editContent = '';
	}

	function handleEditKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			confirmEdit();
		}
		if (e.key === 'Escape') cancelEdit();
	}

	function confirmDelete() {
		onDelete(menuMsgId);
		menuOpen = false;
	}

	$effect(() => {
		if (messages.length && messagesEl) {
			messagesEl.scrollTop = messagesEl.scrollHeight;
		}
	});

	function formatTime(iso: string): string {
		if (!iso) return '';
		const date = new Date(iso);
		if (isNaN(date.getTime())) return iso;
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}
</script>

{#if room}
	<div class="chat-area">
		<div class="chat-header">
			<h2>{roomName ?? room.name ?? 'Direct Message'}</h2>
			{#if room.is_group}
				<span class="chat-meta">{memberCount} members</span>
			{/if}
		</div>

		{#if connectionState === 'reconnecting' || connectionState === 'offline'}
			<div class="connection-banner" role="status">
				{connectionState === 'offline'
					? 'No connection. Messages will not be sent until the connection is restored.'
					: 'Reconnecting. New messages will appear once the connection is back.'}
			</div>
		{/if}

		<div class="messages" bind:this={messagesEl}>
			{#each messages as msg (msg.id)}
				{@const isOwn = msg.sender_id === currentUserId}
				{@const author = getParticipant(msg.sender_id)}
				<div class="message" class:own={isOwn}>
					{#if !isOwn && author}
						<div class="avatar" style="background-color: {author.color}">
							{getInitials(author.name)}
						</div>
					{/if}
					<div class="message-body">
						{#if !isOwn && author}
							<span class="message-author" style="color: {author.color}">
								{author.name}
							</span>
						{/if}
						{#if isOwn}
							<div class="message-wrapper">
								{#if editingId !== msg.id}
									<button
										class="msg-menu-btn"
										aria-label="Message options"
										onclick={(e) => openMenu(e, msg.id)}
									>
										<EllipsisVertical size={14} />
									</button>
								{/if}
								{#if editingId === msg.id}
									<div class="edit-area">
										<Textarea bind:value={editContent} onkeydown={handleEditKeydown} />
										<div class="edit-actions">
											<Button size="sm" variant="ghost" onclick={cancelEdit}>Cancel</Button>
											<Button color="secondary" size="sm" onclick={confirmEdit}>Save</Button>
										</div>
									</div>
								{:else}
									<div class="message-bubble own">{msg.content}</div>
								{/if}
							</div>
						{:else}
							<div class="message-bubble">{msg.content}</div>
						{/if}
						<span class="message-time">
							{formatTime(msg.created_at)}{#if msg.is_edited}<span class="edited-tag">(edited)</span
								>{/if}
						</span>
					</div>
				</div>
			{/each}
		</div>

		<div class="message-input">
			<div class="textarea-wrap">
				<textarea
					class="chat-textarea"
					bind:this={textareaEl}
					bind:value={input}
					placeholder="Type a message..."
					oninput={autoResize}
					onkeydown={handleKeydown}
					rows="1"></textarea>
			</div>
			<Button onclick={send} disabled={!input.trim()}>Send</Button>
		</div>
	</div>

	<Dropdown
		bind:open={menuOpen}
		anchor={menuAnchor}
		matchAnchorWidth={false}
		class="msg-dropdown"
		placement="left"
	>
		<Col>
			<Button variant="ghost" size="sm" onclick={startEdit}>
				<Pencil size={14} />
				Edit
			</Button>
			<Button variant="ghost" size="sm" color="danger" onclick={confirmDelete}>
				<Trash2 size={14} />
				Delete
			</Button>
		</Col>
	</Dropdown>
{:else if pendingUser}
	<div class="chat-area">
		<div class="chat-header">
			<h2>{pendingUser.username}</h2>
		</div>

		<div class="messages pending-messages">
			<div class="pending-hint">
				<div
					class="pending-avatar"
					style="background-color: {colorVar(pendingUser.color || '--primary')}"
				>
					{getInitials(pendingUser.username)}
				</div>
				<div>
					<p>
						This is the start of your conversation with <strong>{pendingUser.username}</strong>.
					</p>
					<p class="pending-note">Send a message to start the conversation.</p>
				</div>
			</div>
		</div>

		<div class="message-input">
			<div class="textarea-wrap">
				<textarea
					class="chat-textarea"
					bind:this={textareaEl}
					bind:value={input}
					placeholder="Type a message..."
					oninput={autoResize}
					onkeydown={handleKeydown}
					rows="1"></textarea>
			</div>
			<Button onclick={send} disabled={!input.trim()}>Send</Button>
		</div>
	</div>
{:else}
	<div class="chat-empty">
		<div class="empty-icon">💬</div>
		<p>Select a conversation to start chatting</p>
	</div>
{/if}

<style>
	.chat-area {
		flex: 1;
		display: flex;
		flex-direction: column;
		height: 100%;
		min-width: 0;
	}

	.chat-header {
		padding: 0.85rem 1.25rem;
		border-bottom: 1px solid var(--border);
		display: flex;
		align-items: baseline;
		gap: 0.75rem;
		background-color: var(--surface);
	}

	.chat-header h2 {
		font-size: 1rem;
		font-weight: 600;
		color: var(--primary-text);
	}

	.chat-meta {
		font-size: 0.8rem;
		color: var(--secondary-text);
	}

	.connection-banner {
		background-color: var(--surface-hover);
		color: var(--secondary-text);
		font-size: 0.78rem;
		padding: 0.4rem 1.25rem;
		border-bottom: 1px solid var(--border);
		flex-shrink: 0;
	}

	.messages {
		flex: 1;
		overflow-y: auto;
		padding: 1rem 1.25rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.message {
		display: flex;
		gap: 0.65rem;
		max-width: 75%;
	}

	.message.own {
		align-self: flex-end;
		flex-direction: row-reverse;
	}

	.avatar {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.8rem;
		font-weight: 600;
		color: var(--bg);
		flex-shrink: 0;
	}

	.message-body {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.message-author {
		font-size: 0.75rem;
		font-weight: 600;
	}

	.message-bubble {
		background-color: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--round-lg);
		padding: 0.5rem 0.85rem;
		font-size: 0.9rem;
		line-height: 1.45;
		color: var(--primary-text);
		word-break: break-word;
	}

	.message-bubble.own {
		background-color: var(--primary);
		border-color: var(--primary);
		color: white;
	}

	.message-time {
		font-size: 0.65rem;
		color: var(--muted-text);
	}

	.message.own .message-time {
		text-align: right;
	}

	.edited-tag {
		font-style: italic;
		opacity: 0.8;
		margin-left: 0.25rem;
	}

	.message-wrapper {
		position: relative;
		display: flex;
		align-items: flex-end;
		gap: 0.25rem;
	}

	.msg-menu-btn {
		background: none;
		border: none;
		color: var(--muted-text);
		cursor: pointer;
		padding: 0.55rem;
		border-radius: var(--round-sm);
		display: flex;
		align-items: center;
		justify-content: center;
		opacity: 0;
		transition:
			opacity 0.15s,
			background-color 0.15s;
		flex-shrink: 0;
		align-self: center;
	}

	.message-wrapper:hover .msg-menu-btn {
		opacity: 1;
	}

	.msg-menu-btn:hover {
		background-color: var(--surface-hover);
		color: var(--primary-text);
	}

	.edit-area {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		background-color: var(--primary);
		border-radius: var(--round-lg);
		padding: 0.5rem 0.75rem;
	}

	.edit-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.25rem;
	}

	.edit-actions :global(.button) {
		color: white;
	}

	.message-input {
		padding: 0.85rem 1.25rem;
		border-top: 1px solid var(--border);
		display: flex;
		align-items: flex-end;
		gap: 0.75rem;
		background-color: var(--surface);
	}

	.textarea-wrap {
		flex: 1;
		display: flex;
	}

	.chat-textarea {
		flex: 1;
		background-color: var(--input);
		border: 1px solid var(--border);
		border-radius: var(--round-lg);
		color: var(--primary-text);
		font: inherit;
		font-size: 0.9rem;
		padding: 0.55rem 0.75rem;
		outline: none;
		resize: none;
		min-height: 40px;
		max-height: 144px;
		overflow-y: auto;
		line-height: 1.45;
		transition: border-color 0.15s;
	}

	.chat-textarea:focus {
		border-color: var(--primary);
	}

	.chat-textarea::placeholder {
		color: var(--muted-text);
	}

	.pending-messages {
		align-items: center;
		justify-content: center;
	}

	.pending-hint {
		display: flex;
		align-items: center;
		gap: 0.85rem;
		max-width: 340px;
		text-align: left;
		color: var(--secondary-text);
		font-size: 0.9rem;
		line-height: 1.5;
	}

	.pending-hint p {
		margin: 0;
	}

	.pending-note {
		font-size: 0.8rem;
		color: var(--muted-text);
		margin-top: 0.25rem;
	}

	.pending-avatar {
		width: 48px;
		height: 48px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 1.15rem;
		font-weight: 600;
		color: var(--bg);
		flex-shrink: 0;
	}

	.chat-empty {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
		color: var(--secondary-text);
	}

	.empty-icon {
		font-size: 3rem;
		opacity: 0.4;
	}

	:global(.msg-dropdown .col-container) {
		gap: 0.15rem;
	}

	:global(.msg-dropdown .button) {
		justify-content: flex-start;
		width: 100%;
	}
</style>
