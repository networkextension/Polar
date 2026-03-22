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
export async function fetchTags(limit = 100, offset = 0) {
    return requestJson(`/api/tags?limit=${limit}&offset=${offset}`);
}
export async function updateTag(id, payload) {
    return requestJson(`/api/tags/${id}`, {
        method: "PUT",
        body: payload,
    });
}
export async function removeTag(id) {
    return requestJson(`/api/tags/${id}`, {
        method: "DELETE",
    });
}
export async function fetchSiteSettings() {
    return requestJson("/api/site-settings");
}
export async function updateSiteSettings(payload) {
    return requestJson("/api/site-settings", {
        method: "PUT",
        body: payload,
    });
}
export async function uploadSiteIcon(formData) {
    return requestJson("/api/site-settings/icon", {
        method: "POST",
        body: formData,
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
