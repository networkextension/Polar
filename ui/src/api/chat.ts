import { request, requestJson } from "./http.js";
import type {
  ChatListResponse,
  ChatMessagesResponse,
  StartChatResponse,
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

export async function fetchMessages(threadId: string, limit = 200) {
  return requestJson<ChatMessagesResponse>(`/api/chats/${threadId}/messages?limit=${limit}`);
}

export async function revokeMessage(threadId: string, messageId: string) {
  return request(`/api/chats/${threadId}/messages/${messageId}`, {
    method: "DELETE",
  });
}

export async function sendMessage(threadId: string, content: string) {
  return request(`/api/chats/${threadId}/messages`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ content }),
  });
}
