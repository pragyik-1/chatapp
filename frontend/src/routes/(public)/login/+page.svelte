<script lang="ts">
  import { goto } from '$app/navigation'
  import { api } from '$lib/api'
  import { Input, Button, Card, toast } from '@hermitk/bluenite'

  let email = $state('')
  let password = $state('')
  let errors = $state<{ email?: string; password?: string }>({})

  function validate() {
    errors = {}
    if (!email.trim()) errors.email = 'Email is required'
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) errors.email = 'Invalid email'
    if (!password) errors.password = 'Password is required'
    else if (password.length < 6) errors.password = 'At least 6 characters'
    return Object.keys(errors).length === 0
  }

  async function handleSubmit(e: Event) {
    e.preventDefault()
    if (!validate()) return
    const { error } = await api.loginUser({ email, password })
    if (error) {
      toast.show({
        variant: 'danger',
        message: error
      })
    } else {
      toast.show({
        variant: 'success',
        message: 'Login successful'
      })
      goto('/')
    }
  }
</script>

<div class="auth-page">
  <Card class="auth-card">
    <div class="auth-header">
      <h1>Welcome back</h1>
      <p>Sign in to your account</p>
    </div>

    <form class="auth-form" onsubmit={handleSubmit}>
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
      <Button type="submit" class="submit-btn">Sign in</Button>
    </form>

    <p class="auth-switch">
      Don't have an account? <a href="/register">Sign up</a>
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
