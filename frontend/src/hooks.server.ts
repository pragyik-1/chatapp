import { api } from '$lib/api'
import { PUBLIC_ROUTES } from '$lib/constants'
import { isJWTValid } from '$lib/server/auth'
import { redirect, type Handle } from '@sveltejs/kit'

export const handle: Handle = async ({ event, resolve }) => {
  event.locals.isAuthenticated = false
  if (PUBLIC_ROUTES.includes(event.url.pathname)) {
    return resolve(event)
  }
  const access_token = event.cookies.get('access_token')
  const refresh_token_hash = event.cookies.get('refresh_token_hash') || ''
  const isValid = isJWTValid(access_token || '')

  if (!isValid && refresh_token_hash) {
    const { data, error } = await api.refreshToken(refresh_token_hash)
    if (error || !data?.access_token || !isJWTValid(data.access_token)) {
      console.error('Refresh failed:', error)
      event.cookies.delete('access_token', { path: '/' })
      event.cookies.delete('refresh_token_hash', { path: '/' })
      throw redirect(303, '/login')
    }
    event.locals.isAuthenticated = true
    event.cookies.set('access_token', data.access_token, {
      path: '/',
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'strict',
    })
  } else if (isValid) {
    event.locals.isAuthenticated = true
  }

  if (!event.locals.isAuthenticated) {
    throw redirect(303, '/login')
  }

  return resolve(event)
}
