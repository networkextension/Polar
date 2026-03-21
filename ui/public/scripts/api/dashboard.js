import { request, requestJson } from "./http.js";
export async function fetchLoginHistory(limit = 5) {
    return requestJson(`/api/login-history?limit=${limit}`);
}
export async function fetchEntries(offset, limit = 10) {
    return requestJson(`/api/markdown?limit=${limit}&offset=${offset}`);
}
export async function fetchEntry(id) {
    return requestJson(`/api/markdown/${id}`);
}
export async function createTag(payload) {
    return requestJson("/api/tags", {
        method: "POST",
        body: payload,
    });
}
export async function deleteEntry(id) {
    return request(`/api/markdown/${id}`, {
        method: "DELETE",
    });
}
export async function uploadUserIcon(formData) {
    return requestJson("/api/user/icon", {
        method: "POST",
        body: formData,
    });
}
export async function beginPasskeyRegistration() {
    return requestJson("/api/passkey/register/begin", {
        method: "POST",
    });
}
export async function finishPasskeyRegistration(sessionId, payload) {
    return requestJson("/api/passkey/register/finish", {
        method: "POST",
        headers: {
            "X-Passkey-Session": sessionId,
        },
        body: payload,
    });
}
