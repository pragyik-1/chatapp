<script lang="ts">
	import { Col } from '@hermitk/bluenite';
	import { api } from '$lib/api';
	import { colorVar, getInitials } from '$lib/utils';
	import type { Room, UserSearchResult } from '$lib/types';

	let {
		rooms,
		selectedRoomId,
		onSelect,
		onStartDM,
		roomNames = {},
		roomColors = {},
	}: {
		rooms: Room[];
		selectedRoomId: string | null;
		onSelect: (room: Room) => void;
		onStartDM: (user: UserSearchResult) => void;
		roomNames?: Record<string, string>;
		roomColors?: Record<string, string>;
	} = $props();

	let searchQuery = $state('');
	let results = $state<UserSearchResult[]>([]);
	let searching = $state(false);
	let searchError = $state('');
	let searchSeq = $state(0);

	let debounceTimer: ReturnType<typeof setTimeout> | undefined;

	function resetSearch() {
		searchSeq++;
		searching = false;
		results = [];
		searchError = '';
	}

	function handleInput() {
		clearTimeout(debounceTimer);
		if (!searchQuery.trim()) {
			resetSearch();
			return;
		}
		searching = true;
		debounceTimer = setTimeout(runSearch, 300);
	}

	async function runSearch() {
		const q = searchQuery.trim();
		if (!q) return;

		const seq = ++searchSeq;
		const res = await api.searchUsers(q);
		if (seq !== searchSeq) return; // stale response, a newer search is in flight

		searching = false;
		if (res.error) {
			searchError = res.error;
			results = [];
			return;
		}
		results = res.data || [];
	}

	function startDM(user: UserSearchResult) {
		searchQuery = '';
		resetSearch();
		onStartDM(user);
	}
</script>

<aside class="sidebar">
	<div class="search-wrap">
		<svg
			class="search-icon"
			width="16"
			height="16"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
		>
			<circle cx="11" cy="11" r="8" />
			<path d="m21 21-4.35-4.35" />
		</svg>
		<input
			class="search-input"
			type="text"
			placeholder="Search users…"
			aria-label="Search users by name, email, or ID"
			bind:value={searchQuery}
			oninput={handleInput}
		/>

		{#if searchQuery.trim()}
			{#if !searching}
				<button class="search-clear" aria-label="Clear search" onclick={resetSearch}>✕</button>
			{/if}
		{/if}
	</div>

	{#if searchQuery.trim()}
		<div class="search-results">
			{#if searching}
				<div class="search-empty">Searching…</div>
			{:else if searchError}
				<div class="search-empty search-error">{searchError}</div>
			{:else if results.length === 0}
				<div class="search-empty">No users found</div>
			{:else}
				{#each results as result (result.id)}
					<button class="search-result" onclick={() => startDM(result)}>
						<span
							class="result-avatar"
							style="background-color: {colorVar(result.color || '--primary')}"
						>
							{getInitials(result.username)}
						</span>
						<span class="result-info">
							<span class="result-name">{result.username}</span>
						</span>
						<span class="result-action" title="Start a direct message">
							<svg
								width="14"
								height="14"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
							>
								<path
									d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"
								/>
							</svg>
						</span>
					</button>
				{/each}
			{/if}
		</div>
	{/if}

	<div class="sidebar-scroll">
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
						{:else if roomColors[room.id]}
							<div class="dm-avatar" style="background-color: {colorVar(roomColors[room.id])}">
								{getInitials(roomNames[room.id] ?? '??')}
							</div>
						{:else}
							<div class="dm-dot"></div>
						{/if}
					</div>
					<div class="conversation-info">
						<span class="conversation-name"
							>{roomNames[room.id] ?? room.name ?? 'Direct Message'}</span
						>
					</div>
				</button>
			{/each}
		</Col>
	</div>
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

	.search-wrap {
		position: relative;
		display: flex;
		align-items: center;
		padding: 0.75rem 0.75rem 0.5rem;
		flex-shrink: 0;
	}

	.search-icon {
		position: absolute;
		left: 1.1rem;
		top: 50%;
		transform: translateY(-50%);
		color: var(--muted-text);
		pointer-events: none;
		z-index: 1;
	}

	.search-input {
		width: 100%;
		background-color: var(--input);
		border: 1px solid var(--border);
		border-radius: var(--round-lg);
		color: var(--primary-text);
		font: inherit;
		font-size: 0.85rem;
		padding: 0.5rem 2rem 0.5rem 2.15rem;
		outline: none;
		transition: border-color 0.15s;
	}

	.search-input:focus {
		border-color: var(--primary);
	}

	.search-input::placeholder {
		color: var(--muted-text);
	}

	.search-clear {
		position: absolute;
		right: 1rem;
		top: 50%;
		transform: translateY(-50%);
		background: none;
		border: none;
		color: var(--muted-text);
		cursor: pointer;
		padding: 0.15rem 0.3rem;
		border-radius: var(--round-sm);
		font-size: 0.7rem;
	}

	.search-clear:hover {
		color: var(--primary-text);
		background-color: var(--surface-hover);
	}

	.search-results {
		flex-shrink: 0;
		max-height: 40%;
		overflow-y: auto;
		border-bottom: 1px solid var(--border);
		padding: 0.25rem 0.5rem 0.5rem;
	}

	.search-empty {
		padding: 0.75rem 0.5rem;
		font-size: 0.8rem;
		color: var(--secondary-text);
		text-align: center;
	}

	.search-empty.search-error {
		color: var(--danger, #e5484d);
	}

	.search-result {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		width: 100%;
		padding: 0.5rem 0.6rem;
		border: none;
		border-radius: var(--round-md);
		background: transparent;
		color: inherit;
		cursor: pointer;
		text-align: left;
		font: inherit;
		transition: background-color 0.15s;
	}

	.search-result:hover,
	.search-result:focus-visible {
		background-color: var(--surface-hover);
		outline: none;
	}

	.result-avatar {
		width: 30px;
		height: 30px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--bg);
		flex-shrink: 0;
	}

	.result-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
	}

	.result-name {
		font-size: 0.85rem;
		font-weight: 500;
		color: var(--primary-text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.result-action {
		display: flex;
		align-items: center;
		color: var(--muted-text);
		flex-shrink: 0;
	}

	.sidebar-scroll {
		flex: 1;
		overflow-y: auto;
		min-height: 0;
	}

	:global(.sidebar-scroll .col-container) {
		overflow-y: auto;
		flex: 1;
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

	.dm-avatar {
		width: 36px;
		height: 36px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.72rem;
		font-weight: 600;
		letter-spacing: 0.02em;
		color: var(--bg);
		flex-shrink: 0;
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
