import type { User } from "./types"

export const auth = $state({
  isAuthenticated: false,
  currentUser: null as User | null,
})
