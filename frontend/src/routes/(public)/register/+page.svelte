<script lang="ts">
  import { api } from '$lib/api'
  import { COLOR_PALETTE } from '$lib/constants'
  import { Input, Button, Card, toast } from '@hermitk/bluenite'
  import { goto } from '$app/navigation'
  import { resolve } from '$app/paths';

  let username = $state('')
  let email = $state('')
  let password = $state('')
  let confirmPassword = $state('')
  let color = $state(COLOR_PALETTE[0])
  let error = $state('')
  let errors = $state<{
    username?: string
    email?: string
    password?: string
    confirmPassword?: string
  }>({})

  function validate() {
    errors = {}
    if (!username.trim()) errors.username = 'Username is required'
    else if (username.length < 3) errors.username = 'At least 3 characters'
    if (!email.trim()) errors.email = 'Email is required'
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) errors.email = 'Invalid email'
    if (!password) errors.password = 'Password is required'
    else if (password.length < 6) errors.password = 'At least 6 characters'
    if (password !== confirmPassword) errors.confirmPassword = 'Passwords do not match'
    return Object.keys(errors).length === 0
  }

  async function handleSubmit(e: Event) {
    e.preventDefault()
    if (!validate()) return
    color = COLOR_PALETTE[Math.floor(Math.random() * COLOR_PALETTE.length)]
    const { error } = await api.registerUser({ username, email, password, color })
    if (error) {
      toast.show({
        variant: 'danger',
        message: error
      })
    } else {
      toast.show({
        variant: 'success',
        message: 'Account created successfully'
      })
      goto(resolve('/'))
    }
  }
</script>

<div class="auth-page">
  <Card class="auth-card">
    <div class="auth-header">
      <h1>Create account</h1>
      <p>Sign up to get started</p>
    </div>

    <form class="auth-form" onsubmit={handleSubmit}>
      <Input
        label="Username"
        placeholder="johndoe"
        bind:value={username}
        error={errors.username}
      />
      <Input
        label="Email"
        type="email"
        placeholder="you@example.com"
        bind:value={email}
        error={errors.email}
      />
      <Input
        label="Password"
        type="password"
        placeholder="••••••••"
        bind:value={password}
        error={errors.password}
      />
      <Input
        label="Confirm password"
        type="password"
        placeholder="••••••••"
        bind:value={confirmPassword}
        error={errors.confirmPassword}
      />
      {#if error}
        <p class="form-error">{error}</p>
      {/if}
      <Button type="submit" class="submit-btn">Create account</Button>
    </form>

    <p class="auth-switch">
      Already have an account? <a href="/login">Sign in</a>
    </p>
  </Card>
</div>

<style>
  .auth-page {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: calc(100vh - 52px);
    padding: 1rem;
  }

  :global(.auth-card) {
    width: 100%;
    max-width: 400px;
  }

  .auth-header {
    text-align: center;
    margin-bottom: 1.5rem;
  }

  .auth-header h1 {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--primary-text);
    margin: 0 0 0.35rem;
  }

  .auth-header p {
    font-size: 0.9rem;
    color: var(--secondary-text);
    margin: 0;
  }

  .auth-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  :global(.submit-btn) {
    width: 100%;
    margin-top: 0.5rem;
  }

  .form-error {
    color: var(--danger);
    font-size: 0.85rem;
    margin: 0;
  }

  .auth-switch {
    text-align: center;
    font-size: 0.85rem;
    color: var(--secondary-text);
    margin: 1.25rem 0 0;
  }

  .auth-switch a {
    color: var(--primary);
    text-decoration: none;
  }

  .auth-switch a:hover {
    text-decoration: underline;
  }
</style>
