import { requireAuth } from '$lib/server/auth'

export const load = ({ locals }) => {
  requireAuth(locals)
}
