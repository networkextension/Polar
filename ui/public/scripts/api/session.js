import { request, requestJson } from "./http.js";
export async function fetchCurrentUser() {
    return requestJson("/api/me");
}
export async function logout() {
    return request("/api/logout", { method: "POST" });
}
export async function sendEmailVerification() {
    return requestJson("/api/email-verification/send", {
        method: "POST",
    });
}
