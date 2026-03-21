import { fetchChats, fetchMessages, revokeMessage as revokeChatMessage, sendMessage, startChat } from "./api/chat.js";
import { fetchCurrentUser } from "./api/session.js";
import { resolveAvatar } from "./lib/avatar.js";
import { byId } from "./lib/dom.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";
import type { ChatEventPayload, ChatMessage, ChatSummary } from "./types/chat.js";
const chatWelcome = byId<HTMLElement>("chatWelcome");
const chatList = byId<HTMLElement>("chatList");
const chatTitle = byId<HTMLElement>("chatTitle");
const chatSubtitle = byId<HTMLElement>("chatSubtitle");
const messageList = byId<HTMLElement>("messageList");
const messageForm = byId<HTMLFormElement>("messageForm");
const messageInput = byId<HTMLInputElement>("messageInput");
const chatRefreshBtn = byId<HTMLButtonElement>("chatRefreshBtn");

let currentUserId = "";
let activeThreadId: string | null = null;
let chatCache: ChatSummary[] = [];
let pollTimer: number | null = null;
let ws: WebSocket | null = null;
let wsConnected = false;

initStoredTheme();
bindThemeSync();

function formatTime(value?: string): string {
  if (!value) {
    return "";
  }
  return new Date(value).toLocaleString();
}

function getTargetFromQuery(): { userId: string | null; username: string | null } {
  const params = new URLSearchParams(window.location.search);
  return {
    userId: params.get("user_id"),
    username: params.get("username"),
  };
}

async function loadProfile(): Promise<void> {
  const { response, data } = await fetchCurrentUser();
  if (!response.ok) {
    window.location.href = "/login.html";
    return;
  }
  currentUserId = data.user_id;
  chatWelcome.textContent = `你好，${data.username}，和好友聊两句吧。`;
}

function renderChatList(chats: ChatSummary[]): void {
  chatList.innerHTML = "";
  if (!chats.length) {
    chatList.innerHTML = "<div class='chat-empty'>暂无会话</div>";
    return;
  }

  chats.forEach((chat) => {
    const item = document.createElement("button");
    item.type = "button";
    item.className = "chat-item";
    if (activeThreadId === chat.id) {
      item.classList.add("active");
    }
    const unreadBadge = chat.unread_count
      ? `<span class="unread-badge">${chat.unread_count}</span>`
      : "";
    const avatar = resolveAvatar(chat.other_username, chat.other_user_icon, 48);
    item.innerHTML = `
      <div class="chat-item-header">
        <img class="avatar-xs" src="${avatar}" alt="${chat.other_username}" />
        <div class="chat-item-body">
          <div class="chat-item-title">${chat.other_username}${unreadBadge}</div>
          <div class="chat-item-meta">
            <span>${chat.last_message || "暂无消息"}</span>
            <span>${chat.last_message_at ? formatTime(chat.last_message_at) : ""}</span>
          </div>
        </div>
      </div>
    `;
    item.addEventListener("click", () => {
      void openChat(chat);
    });
    chatList.appendChild(item);
  });
}

async function loadChats(focusThreadId: string | null = null): Promise<void> {
  const { response, data } = await fetchChats();
  if (!response.ok) {
    chatList.innerHTML = "<div class='chat-empty'>无法加载会话</div>";
    return;
  }

  chatCache = data.chats || [];
  if (focusThreadId) {
    const match = chatCache.find((item) => item.id === focusThreadId);
    if (match) {
      activeThreadId = match.id;
    }
  }
  renderChatList(chatCache);
}

async function startChatWithUser(userId: string): Promise<ChatSummary | null> {
  const { response, data } = await startChat(userId);
  if (!response.ok) {
    chatSubtitle.textContent = data.error || "无法创建会话";
    return null;
  }
  return data.chat || null;
}

function renderMessages(messages: ChatMessage[]): void {
  if (!messages.length) {
    messageList.innerHTML = "<div class='chat-empty'>暂无消息</div>";
    return;
  }

  messageList.innerHTML = messages
    .map((msg) => {
      const isMine = msg.sender_id === currentUserId;
      const content = msg.deleted ? "消息已撤回" : msg.content;
      const bubbleClass = msg.deleted ? "message-bubble deleted" : "message-bubble";
      const revokeButton =
        isMine && !msg.deleted
          ? `<button class="message-revoke" data-id="${msg.id}" type="button">撤回</button>`
          : "";
      const avatar = resolveAvatar(msg.sender_username, msg.sender_icon, 48);
      return `
        <div class="message-item ${isMine ? "mine" : "other"}">
          <div class="message-head">
            <img class="avatar-xs" src="${avatar}" alt="${msg.sender_username}" />
            <div class="message-meta">${msg.sender_username} · ${formatTime(msg.created_at)}</div>
          </div>
          <div class="message-row">
            <div class="${bubbleClass}">${content}</div>
            ${revokeButton}
          </div>
        </div>
      `;
    })
    .join("");

  messageList.querySelectorAll<HTMLButtonElement>(".message-revoke").forEach((button) => {
    button.addEventListener("click", async () => {
      const messageId = button.dataset.id;
      if (!messageId) {
        return;
      }
      await revokeMessage(messageId);
    });
  });
  messageList.scrollTop = messageList.scrollHeight;
}

async function loadMessages(threadId: string): Promise<void> {
  messageList.innerHTML = "<div class='chat-empty'>加载中...</div>";
  const { response, data } = await fetchMessages(threadId);
  if (!response.ok) {
    messageList.innerHTML = "<div class='chat-empty'>无法加载消息</div>";
    return;
  }
  renderMessages(data.messages || []);
}

async function openChat(chat: ChatSummary): Promise<void> {
  activeThreadId = chat.id;
  chatTitle.textContent = chat.other_username;
  chatSubtitle.textContent = `正在与 ${chat.other_username} 私聊`;
  messageInput.disabled = false;
  renderChatList(chatCache);
  await loadMessages(chat.id);
  await loadChats(chat.id);
}

async function revokeMessage(messageId: string): Promise<void> {
  if (!activeThreadId) {
    return;
  }
  const response = await revokeChatMessage(activeThreadId, messageId);
  if (!response.ok) {
    return;
  }
  await loadMessages(activeThreadId);
  await loadChats(activeThreadId);
}

function startPolling(): void {
  if (wsConnected) {
    return;
  }
  if (pollTimer) {
    window.clearInterval(pollTimer);
  }
  pollTimer = window.setInterval(async () => {
    await loadChats(activeThreadId);
    if (activeThreadId) {
      await loadMessages(activeThreadId);
    }
  }, 5000);
}

function stopPolling(): void {
  if (pollTimer) {
    window.clearInterval(pollTimer);
    pollTimer = null;
  }
}

function getWebSocketUrl(): string {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/ws/chat`;
}

function connectWebSocket(): void {
  try {
    ws = new WebSocket(getWebSocketUrl());
  } catch {
    startPolling();
    return;
  }

  ws.addEventListener("open", () => {
    wsConnected = true;
    stopPolling();
  });

  ws.addEventListener("close", () => {
    wsConnected = false;
    startPolling();
  });

  ws.addEventListener("error", () => {
    wsConnected = false;
    startPolling();
  });

  ws.addEventListener("message", async (event) => {
    let payload: ChatEventPayload;
    try {
      payload = JSON.parse(event.data);
    } catch {
      return;
    }

    if (!payload?.type) {
      return;
    }

    const chatId = payload.chat_id;
    if (payload.type === "message") {
      if (activeThreadId === chatId && chatId) {
        await loadMessages(chatId);
      }
      await loadChats(activeThreadId);
      return;
    }

    if (payload.type === "read") {
      await loadChats(activeThreadId);
      return;
    }

    if (payload.type === "revoke") {
      if (activeThreadId === chatId && chatId) {
        await loadMessages(chatId);
      }
      await loadChats(activeThreadId);
    }
  });
}

messageForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!activeThreadId) {
    return;
  }

  const content = messageInput.value.trim();
  if (!content) {
    return;
  }

  const response = await sendMessage(activeThreadId, content);
  if (!response.ok) {
    return;
  }

  messageInput.value = "";
  await loadMessages(activeThreadId);
  await loadChats(activeThreadId);
});

chatRefreshBtn.addEventListener("click", async () => {
  await loadChats(activeThreadId);
  if (activeThreadId) {
    await loadMessages(activeThreadId);
  }
});

async function init(): Promise<void> {
  await loadProfile();
  messageInput.disabled = true;
  connectWebSocket();

  const target = getTargetFromQuery();
  if (target.userId) {
    if (target.userId === currentUserId) {
      chatSubtitle.textContent = "不能和自己聊天";
      await loadChats();
      startPolling();
      return;
    }
    const chat = await startChatWithUser(target.userId);
    await loadChats(chat ? chat.id : null);
    if (chat) {
      await openChat(chat);
    }
    startPolling();
    return;
  }

  await loadChats();
  startPolling();
}

void init();
