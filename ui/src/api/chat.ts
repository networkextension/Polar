import { request, requestJson } from "./http.js";
import type {
  ChatListResponse,
  SharedMarkdownResponse,
  ChatMessagesResponse,
  StartChatResponse,
  LLMThreadListResponse,
  ChatLLMConfigListResponse,
} from "../types/chat.js";

export async function fetchChats(limit = 50) {
  return requestJson<ChatListResponse>(`/api/chats?limit=${limit}`);
}

export async function startChat(userId: string) {
  return requestJson<StartChatResponse>("/api/chats/start", {
    method: "POST",
    body: { user_id: userId },
  });
}

export async function fetchMessages(threadId: string, limit = 200, llmThreadId?: number | null) {
  const params = new URLSearchParams({ limit: String(limit) });
  if (llmThreadId) {
    params.set("llm_thread_id", String(llmThreadId));
  }
  return requestJson<ChatMessagesResponse>(`/api/chats/${threadId}/messages?${params.toString()}`);
}

export async function fetchLLMThreads(threadId: string, activeThreadId?: number | null) {
  const params = new URLSearchParams();
  if (activeThreadId) {
    params.set("active_thread_id", String(activeThreadId));
  }
  const suffix = params.toString() ? `?${params.toString()}` : "";
  return requestJson<LLMThreadListResponse>(`/api/chats/${threadId}/llm-threads${suffix}`);
}

export async function createLLMThread(threadId: string, title = "") {
  return requestJson<LLMThreadListResponse>(`/api/chats/${threadId}/llm-threads`, {
    method: "POST",
    body: { title },
  });
}

export async function updateLLMThread(threadId: string, llmThreadId: number, title: string) {
  return requestJson<LLMThreadListResponse>(`/api/chats/${threadId}/llm-threads/${llmThreadId}`, {
    method: "PUT",
    body: { title },
  });
}

export async function switchLLMThreadConfig(threadId: string, llmThreadId: number, llmConfigId: number) {
  return requestJson<LLMThreadListResponse>(`/api/chats/${threadId}/llm-threads/${llmThreadId}/config`, {
    method: "PUT",
    body: { llm_config_id: llmConfigId },
  });
}

export async function fetchChatLLMConfigs() {
  return requestJson<ChatLLMConfigListResponse>("/api/llm-configs/available");
}

export async function fetchSharedMarkdown(threadId: string, messageId: string) {
  return requestJson<SharedMarkdownResponse>(`/api/chats/${threadId}/messages/${messageId}/markdown`);
}

export async function revokeMessage(threadId: string, messageId: string) {
  return request(`/api/chats/${threadId}/messages/${messageId}`, {
    method: "DELETE",
  });
}

export async function retryMessage(threadId: string, messageId: string) {
  return requestJson<{ message?: string; content?: string; error?: string }>(`/api/chats/${threadId}/messages/${messageId}/retry`, {
    method: "POST",
  });
}

export async function sendMessage(threadId: string, content: string, llmThreadId?: number | null) {
  return request(`/api/chats/${threadId}/messages`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ content, llm_thread_id: llmThreadId || undefined }),
  });
}
