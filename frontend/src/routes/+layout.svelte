<script lang="ts">
  import '../app.css'
  import { NavBar, Button, ToastManager, toggleTheme } from '@hermitk/bluenite'
  import { theme } from '@hermitk/bluenite'
  import favicon from '$lib/assets/favicon.svg'

  let { children } = $props()
</script>

<svelte:head>
  <link rel="icon" href={favicon} />
</svelte:head>

<NavBar behavior="static" height="52px" mobile={false}>
  <span class="app-title">ChatApp</span>
  <Button variant="ghost" size="sm" onclick={toggleTheme}>
    {#if theme.value === 'dark'}
      <svg
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <circle cx="12" cy="12" r="5" /><path
          d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"
        />
      </svg>
    {:else}
      <svg
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
      </svg>
    {/if}
  </Button>
</NavBar>

{@render children()}
<ToastManager />

<style>
  .app-title {
    font-size: 1.25rem;
    font-weight: bold;
    color: var(--color-text);
  }
  :global(.navbar-inner) {
    max-width: 100%;
  }
</style>
