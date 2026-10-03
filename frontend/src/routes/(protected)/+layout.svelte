<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';
	import { realtime } from '$lib/realtime';

	let { children }: { children: Snippet } = $props();

	// The socket is owned here, not by the chat page, so it stays open across
	// client-side navigation and closing it on unmount happens exactly once.
	onMount(() => {
		void realtime.connect();
		return () => realtime.disconnect();
	});
</script>

{@render children()}
