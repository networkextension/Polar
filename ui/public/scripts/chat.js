import { fetchChats, fetchMessages, revokeMessage as revokeChatMessage, sendMessage, startChat } from "./api/chat.js";
import { fetchCurrentUser } from "./api/session.js";
import { resolveAvatar } from "./lib/avatar.js";
import { byId } from "./lib/dom.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";
const chatWelcome = byId("chatWelcome");
const chatList = byId("chatList");
const chatTitle = byId("chatTitle");
const chatSubtitle = byId("chatSubtitle");
const messageList = byId("messageList");
const messageForm = byId("messageForm");
const messageInput = byId("messageInput");
const chatRefreshBtn = byId("chatRefreshBtn");
let currentUserId = "";
let activeThreadId = null;
let chatCache = [];
let pollTimer = null;
let ws = null;
let wsConnected = false;
initStoredTheme();
bindThemeSync();
function formatTime(value) {
    if (!value) {
        return "";
    }
    return new Date(value).toLocaleString();
}
function getTargetFromQuery() {
    const params = new URLSearchParams(window.location.search);
    return {
        userId: params.get("user_id"),
        username: params.get("username"),
    };
}
async function loadProfile() {
    const { response, data } = await fetchCurrentUser();
    if (!response.ok) {
        window.location.href = "/login.html";
        return;
    }
    currentUserId = data.user_id;
    chatWelcome.textContent = `你好，${data.username}，和好友聊两句吧。`;
}
function renderChatList(chats) {
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
async function loadChats(focusThreadId = null) {
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
async function startChatWithUser(userId) {
    const { response, data } = await startChat(userId);
    if (!response.ok) {
        chatSubtitle.textContent = data.error || "无法创建会话";
        return null;
    }
    return data.chat || null;
}
function renderMessages(messages) {
    if (!messages.length) {
        messageList.innerHTML = "<div class='chat-empty'>暂无消息</div>";
        return;
    }
    messageList.innerHTML = messages
        .map((msg) => {
        const isMine = msg.sender_id === currentUserId;
        const content = msg.deleted ? "消息已撤回" : msg.content;
        const bubbleClass = msg.deleted ? "message-bubble deleted" : "message-bubble";
        const revokeButton = isMine && !msg.deleted
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
    messageList.querySelectorAll(".message-revoke").forEach((button) => {
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
async function loadMessages(threadId) {
    messageList.innerHTML = "<div class='chat-empty'>加载中...</div>";
    const { response, data } = await fetchMessages(threadId);
    if (!response.ok) {
        messageList.innerHTML = "<div class='chat-empty'>无法加载消息</div>";
        return;
    }
    renderMessages(data.messages || []);
}
async function openChat(chat) {
    activeThreadId = chat.id;
    chatTitle.textContent = chat.other_username;
    chatSubtitle.textContent = `正在与 ${chat.other_username} 私聊`;
    messageInput.disabled = false;
    renderChatList(chatCache);
    await loadMessages(chat.id);
    await loadChats(chat.id);
}
async function revokeMessage(messageId) {
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
function startPolling() {
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
function stopPolling() {
    if (pollTimer) {
        window.clearInterval(pollTimer);
        pollTimer = null;
    }
}
function getWebSocketUrl() {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    return `${protocol}//${window.location.host}/ws/chat`;
}
function connectWebSocket() {
    try {
        ws = new WebSocket(getWebSocketUrl());
    }
    catch {
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
        let payload;
        try {
            payload = JSON.parse(event.data);
        }
        catch {
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
async function init() {
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
