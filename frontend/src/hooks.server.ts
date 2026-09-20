import { api } from "$lib/api";
import { PUBLIC_ROUTES } from "$lib/constants"
import { isJWTValid } from "$lib/server/auth";
import { redirect, type Handle } from "@sveltejs/kit";

export const handle: Handle = async ({ event, resolve }) => {
    event.locals.isAuthenticated = false;
    if (PUBLIC_ROUTES.includes(event.url.pathname)) {
        return resolve(event)
    }
    const token = event.cookies.get("token")
    if (!token) {
        throw redirect(303, "/login")
    }

    const isValid = isJWTValid(token);
    event.locals.isAuthenticated = isValid;
    if (!isValid) {
        const { data, error } = await api.refreshToken();
        if (error) {
            console.error("Error refreshing token:", error);
            event.cookies.delete("access_token", { path: "/" });
            throw redirect(303, "/login");
        }

        if (data) {
            event.locals.isAuthenticated = true;
            event.cookies.set("access_token", data.access_token, {
                path: "/",
                secure: true,
                sameSite: "strict"
            });
        } else {
            event.cookies.delete("access_token", { path: "/" });
            throw redirect(303, "/login");
        }
    }

    if (!event.locals.isAuthenticated) {
        throw redirect(303, "/login")
    }

    return resolve(event)
}