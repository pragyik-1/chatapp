<script lang="ts">
	import { Col } from '@hermitk/bluenite';
	import type { Room } from '$lib/types';

	let {
		rooms,
		selectedRoomId,
		onSelect,
	}: {
		rooms: Room[];
		selectedRoomId: string | null;
		onSelect: (room: Room) => void;
	} = $props();
</script>

<aside class="sidebar">
	<Col gap={0.25}>
		{#each rooms as room (room.id)}
			<button
				class="conversation-item"
				class:selected={room.id === selectedRoomId}
				onclick={() => onSelect(room)}
			>
				<div class="conversation-icon" class:group={room.is_group}>
					{#if room.is_group}
						<svg
							width="18"
							height="18"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
						>
							<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
							<circle cx="9" cy="7" r="4" />
							<path d="M23 21v-2a4 4 0 0 0-3-3.87" />
							<path d="M16 3.13a4 4 0 0 1 0 7.75" />
						</svg>
					{:else}
						<div class="dm-dot"></div>
					{/if}
				</div>
				<div class="conversation-info">
					<span class="conversation-name">{room.name ?? 'Direct Message'}</span>
				</div>
			</button>
		{/each}
	</Col>
</aside>

<style>
	.sidebar {
		width: 300px;
		min-width: 300px;
		height: 100%;
		background-color: var(--surface);
		border-right: 1px solid var(--border);
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	:global(.col-container) {
		flex: 1;
		overflow-y: auto;
		padding: 0.5rem;
	}

	.conversation-item {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		width: 100%;
		padding: 0.65rem 0.75rem;
		border: none;
		border-radius: var(--round-md);
		background: transparent;
		color: inherit;
		cursor: pointer;
		text-align: left;
		font: inherit;
		transition: background-color 0.15s;
		border-left: 3px solid transparent;
	}

	.conversation-item:hover {
		background-color: var(--surface-hover);
	}

	.conversation-item.selected {
		background-color: var(--surface-hover);
		border-left-color: var(--primary);
	}

	.conversation-icon {
		width: 36px;
		height: 36px;
		border-radius: 50%;
		background-color: var(--input);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		color: var(--secondary-text);
	}

	.conversation-icon.group {
		background-color: color-mix(in srgb, var(--primary) 20%, transparent);
		color: var(--primary);
	}

	.dm-dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		background-color: var(--success);
	}

	.conversation-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}

	.conversation-name {
		font-size: 0.9rem;
		font-weight: 500;
		color: var(--primary-text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
</style>
