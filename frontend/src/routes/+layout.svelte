<script lang="ts">
  import '../app.css'
  import { NavBar, Button, ToastManager, toggleTheme, Row } from '@hermitk/bluenite'
  import { theme } from '@hermitk/bluenite'
  import favicon from '$lib/assets/favicon.svg'
  import { HouseIcon, MoonIcon, Settings2Icon, SettingsIcon, SunIcon } from 'lucide-svelte'
  import { goto } from '$app/navigation'

  let { children } = $props()
</script>

<svelte:head>
  <link rel="icon" href={favicon} />
</svelte:head>

<NavBar behavior="static" height="52px" mobile={false}>
  <Button variant="ghost" class="app-title" onclick={() => goto("/")}>
    <HouseIcon size={16} />
    ChatApp
  </Button>
  <Row>
    <Button variant="ghost" size="sm" onclick={toggleTheme}>
      {#if theme.value === 'dark'}
       <SunIcon size={16} />
      {:else}
        <MoonIcon size={16} />
      {/if}
    </Button>
    <Button onclick={() => goto('/settings')} variant="ghost" size="sm">
      <Settings2Icon size={16} />
    </Button>
  </Row>
</NavBar>

{@render children()}
<ToastManager />

<style>
  :global(.navbar-inner) {
    max-width: 100%;
  }
</style>
