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
export async function fetchLLMConfigs() {
    return requestJson("/api/llm-configs");
}
export async function fetchAvailableLLMConfigs() {
    return requestJson("/api/llm-configs/available");
}
export async function createLLMConfig(payload) {
    return requestJson("/api/llm-configs", {
        method: "POST",
        body: payload,
    });
}
export async function testLLMConfig(payload) {
    return requestJson("/api/llm-configs/test", {
        method: "POST",
        body: payload,
    });
}
export async function updateLLMConfig(id, payload) {
    return requestJson(`/api/llm-configs/${id}`, {
        method: "PUT",
        body: payload,
    });
}
export async function removeLLMConfig(id) {
    return requestJson(`/api/llm-configs/${id}`, {
        method: "DELETE",
    });
}
export async function fetchBotUsers() {
    return requestJson("/api/bots");
}
export async function createBotUser(payload) {
    return requestJson("/api/bots", {
        method: "POST",
        body: payload,
    });
}
export async function updateBotUser(id, payload) {
    return requestJson(`/api/bots/${id}`, {
        method: "PUT",
        body: payload,
    });
}
export async function removeBotUser(id) {
    return requestJson(`/api/bots/${id}`, {
        method: "DELETE",
    });
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
export async function uploadApplePushCertificate(environment, formData) {
    return requestJson(`/api/site-settings/apple-push-cert?env=${environment}`, {
        method: "POST",
        body: formData,
    });
}
export async function deleteApplePushCertificate(environment) {
    return requestJson(`/api/site-settings/apple-push-cert?env=${environment}`, {
        method: "DELETE",
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
export async function fetchPasskeys() {
    return requestJson("/api/passkeys");
}
export async function removePasskey(credentialId) {
    return requestJson(`/api/passkeys/${encodeURIComponent(credentialId)}`, {
        method: "DELETE",
    });
}
