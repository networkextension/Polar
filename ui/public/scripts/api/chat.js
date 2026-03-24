import { request, requestJson } from "./http.js";
export async function fetchChats(limit = 50) {
    return requestJson(`/api/chats?limit=${limit}`);
}
export async function startChat(userId) {
    return requestJson("/api/chats/start", {
        method: "POST",
        body: { user_id: userId },
    });
}
export async function fetchMessages(threadId, limit = 200) {
    return requestJson(`/api/chats/${threadId}/messages?limit=${limit}`);
}
export async function fetchSharedMarkdown(threadId, messageId) {
    return requestJson(`/api/chats/${threadId}/messages/${messageId}/markdown`);
}
export async function revokeMessage(threadId, messageId) {
    return request(`/api/chats/${threadId}/messages/${messageId}`, {
        method: "DELETE",
    });
}
export async function sendMessage(threadId, content) {
    return request(`/api/chats/${threadId}/messages`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ content }),
    });
}
