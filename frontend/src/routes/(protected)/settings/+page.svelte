<script lang="ts">
  import { api } from '$lib/api'
  import { COLOR_PALETTE } from '$lib/constants'
  import { colorVar } from '$lib/utils'
  import { Card, Button, theme, toggleTheme, Switch } from '@hermitk/bluenite'
  import { toast } from '@hermitk/bluenite'

  let color = $state(COLOR_PALETTE[0])
  let loading = $state(true)
  let saving = $state(false)

  api.getUserSettings()
    .then((response) => {
      if (response.data) {
        color = response.data.color
      }
    })
    .finally(() => (loading = false))

  async function save() {
    saving = true
    const response = await api.updateUserSettings({
      color,
    })
    if (response.error) {
      toast.show({
        message: response.error,
        variant: 'danger',
      })
    } else if (response.data) {
      color = response.data.color
      toast.show({ message: 'Settings saved', variant: 'success' })
    }
    saving = false
  }
</script>

<div class="settings-page">
  {#if loading}
    <Card class="settings-card">Loading settings…</Card>
  {:else}
    <Card class="settings-card">
      <h1 class="settings-title">Settings</h1>

      <div class="settings-field">
        <span class="settings-label">Color</span>
        <div class="color-picker">
          {#each COLOR_PALETTE as token (token)}
            <button
              type="button"
              class="color-swatch"
              class:selected={color === token}
              style="background-color: {colorVar(token)}"
              aria-label={token}
              onclick={() => (color = token)}
            ></button>
          {/each}
        </div>
      </div>

      <Switch
        label="Dark Mode"
        checked={theme.value === 'dark'}
        onchange={toggleTheme}
      />

      <Button variant="fill" onclick={save} disabled={saving} class="save-btn">
        {saving ? 'Saving…' : 'Save'}
      </Button>
    </Card>
  {/if}
</div>

<style>
  .settings-page {
    display: flex;
    justify-content: center;
    padding: 2rem 1rem;
  }

  :global(.settings-card) {
    width: 100%;
    max-width: 420px;
  }

  .settings-title {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--color-text);
    margin: 0 0 1.5rem;
  }

  .settings-field {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    margin-bottom: 1.25rem;
  }

  .settings-label {
    font-size: 0.85rem;
    font-weight: 500;
    color: var(--color-text);
  }

  .color-picker {
    display: flex;
    gap: 0.6rem;
  }

  .color-swatch {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
    padding: 0;
  }

  .color-swatch:focus-visible {
    outline: 2px solid var(--primary);
    outline-offset: 2px;
  }

  .color-swatch.selected {
    box-shadow: 0 0 0 2px var(--color-bg), 0 0 0 4px var(--color-text);
    transform: scale(1.05);
  }

  :global(.save-btn) {
    width: 100%;
  }
</style>