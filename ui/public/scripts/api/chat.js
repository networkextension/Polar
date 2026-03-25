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
export async function fetchMessages(threadId, limit = 200, llmThreadId) {
    const params = new URLSearchParams({ limit: String(limit) });
    if (llmThreadId) {
        params.set("llm_thread_id", String(llmThreadId));
    }
    return requestJson(`/api/chats/${threadId}/messages?${params.toString()}`);
}
export async function fetchLLMThreads(threadId, activeThreadId) {
    const params = new URLSearchParams();
    if (activeThreadId) {
        params.set("active_thread_id", String(activeThreadId));
    }
    const suffix = params.toString() ? `?${params.toString()}` : "";
    return requestJson(`/api/chats/${threadId}/llm-threads${suffix}`);
}
export async function createLLMThread(threadId, title = "") {
    return requestJson(`/api/chats/${threadId}/llm-threads`, {
        method: "POST",
        body: { title },
    });
}
export async function updateLLMThread(threadId, llmThreadId, title) {
    return requestJson(`/api/chats/${threadId}/llm-threads/${llmThreadId}`, {
        method: "PUT",
        body: { title },
    });
}
export async function switchLLMThreadConfig(threadId, llmThreadId, llmConfigId) {
    return requestJson(`/api/chats/${threadId}/llm-threads/${llmThreadId}/config`, {
        method: "PUT",
        body: { llm_config_id: llmConfigId },
    });
}
export async function fetchChatLLMConfigs() {
    return requestJson("/api/llm-configs/available");
}
export async function fetchSharedMarkdown(threadId, messageId) {
    return requestJson(`/api/chats/${threadId}/messages/${messageId}/markdown`);
}
export async function revokeMessage(threadId, messageId) {
    return request(`/api/chats/${threadId}/messages/${messageId}`, {
        method: "DELETE",
    });
}
export async function retryMessage(threadId, messageId) {
    return requestJson(`/api/chats/${threadId}/messages/${messageId}/retry`, {
        method: "POST",
    });
}
export async function sendMessage(threadId, content, llmThreadId) {
    return request(`/api/chats/${threadId}/messages`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ content, llm_thread_id: llmThreadId || undefined }),
    });
}
