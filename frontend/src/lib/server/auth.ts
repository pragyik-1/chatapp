import { jwtDecode } from "jwt-decode";

export function isJWTValid(token: string): boolean {
    if (!token) return false;
    try {
        const decoded = jwtDecode(token) as { exp: number };
        const currentTime = Math.floor(Date.now() / 1000);
        return decoded.exp > currentTime;
    } catch (error) {
        console.error("Invalid JWT:", error);
        return false;
    }
}