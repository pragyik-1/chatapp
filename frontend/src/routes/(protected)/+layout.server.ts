import { requireAuth } from "$lib/server/auth";

export const load = ({ locals, url }) => {
    console.log("locals.isAuthenticated:", locals.isAuthenticated);
  requireAuth(locals);
}