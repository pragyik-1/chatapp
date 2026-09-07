import { PROTECTED_ROUTES } from "$lib/constants"
import { isJWTValid } from "$lib/server/auth";
import { redirect, type Handle } from "@sveltejs/kit";

export const handle: Handle = async ({ event, resolve }) => {
    if (PROTECTED_ROUTES.includes(event.url.pathname)) {
        const token = event.cookies.get("token")
        if (!token) {
            throw redirect(303, "/login")
        }

        const isValid = isJWTValid(token);
        if (!isValid) {
            throw redirect(303, "/login")
        }
    }
    return resolve(event)
}