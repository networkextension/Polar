import { createLLMThread, fetchChats, fetchLLMThreads, fetchMessages, fetchSharedMarkdown, revokeMessage as revokeChatMessage, sendMessage, startChat } from "./api/chat.js";
import { requestJson } from "./api/http.js";
import { fetchCurrentUser } from "./api/session.js";
import { resolveAvatar } from "./lib/avatar.js";
import { formatDeviceType } from "./lib/client.js";
import { byId } from "./lib/dom.js";
import { renderMarkdown } from "./lib/marked.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";
const chatWelcome = byId("chatWelcome");
const chatList = byId("chatList");
const chatTitle = byId("chatTitle");
const chatSubtitle = byId("chatSubtitle");
const messageList = byId("messageList");
const messageForm = byId("messageForm");
const messageInput = byId("messageInput");
const chatRefreshBtn = byId("chatRefreshBtn");
const chatNewTopicBtn = byId("chatNewTopicBtn");
const chatThreadBar = byId("chatThreadBar");
const chatThreadSelect = byId("chatThreadSelect");
let currentUserId = "";
let activeThreadId = null;
let chatCache = [];
let pollTimer = null;
let ws = null;
let wsConnected = false;
let activeMessages = [];
let activeMessageLoadedAt = "";
let activeLLMThreadId = null;
let activeLLMThreads = [];
let activeIsAIChat = false;
const expandedMarkdownMessages = new Set();
const sharedMarkdownContentCache = new Map();
const sharedMarkdownLoading = new Set();
function escapeHtml(input) {
    return input
        .split("&").join("&amp;")
        .split("<").join("&lt;")
        .split(">").join("&gt;")
        .split('"').join("&quot;")
        .split("'").join("&#39;");
}
async function copyTextToClipboard(text) {
    if (!text) {
        return false;
    }
    if (navigator.clipboard && window.isSecureContext) {
        try {
            await navigator.clipboard.writeText(text);
            return true;
        }
        catch {
            // Fallback below.
        }
    }
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "true");
    textarea.style.position = "fixed";
    textarea.style.top = "-1000px";
    textarea.style.opacity = "0";
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    let copied = false;
    try {
        copied = document.execCommand("copy");
    }
    catch {
        copied = false;
    }
    finally {
        document.body.removeChild(textarea);
    }
    return copied;
}
initStoredTheme();
bindThemeSync();
function formatTime(value) {
    if (!value) {
        return "";
    }
    return new Date(value).toLocaleString();
}
function formatPresence(chat) {
    if (chat.other_user_online) {
        return `在线 · ${formatDeviceType(chat.other_user_device_type)}`;
    }
    if (chat.other_user_last_seen_at) {
        return `离线 · 上次在线 ${formatTime(chat.other_user_last_seen_at)}`;
    }
    return `离线 · 最近设备 ${formatDeviceType(chat.other_user_device_type)}`;
}
function truncatePreview(input, maxLength = 20) {
    const text = (input || "").trim();
    if (!text) {
        return "暂无消息";
    }
    return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text;
}
function updateActiveChatHeader() {
    if (!activeThreadId) {
        return;
    }
    const chat = chatCache.find((item) => item.id === activeThreadId);
    if (!chat) {
        return;
    }
    chatTitle.textContent = chat.other_username;
    chatSubtitle.textContent = formatPresence(chat);
}
function isAIChat(chat) {
    if (!chat) {
        return false;
    }
    return chat.other_user_id === "system" || chat.other_user_id.startsWith("bot_");
}
function renderLLMThreadBar() {
    chatThreadBar.hidden = !activeIsAIChat;
    chatNewTopicBtn.hidden = !activeIsAIChat;
    if (!activeIsAIChat) {
        chatThreadSelect.innerHTML = "";
        return;
    }
    if (!activeLLMThreads.length) {
        chatThreadSelect.innerHTML = `<option value="">默认话题</option>`;
        chatThreadSelect.disabled = true;
        return;
    }
    chatThreadSelect.disabled = false;
    chatThreadSelect.innerHTML = activeLLMThreads
        .map((thread) => `<option value="${thread.id}">${escapeHtml(thread.title || "新话题")}</option>`)
        .join("");
    if (activeLLMThreadId) {
        chatThreadSelect.value = String(activeLLMThreadId);
    }
}
function getMessageMarker(message) {
    if (!message) {
        return "";
    }
    return `${message.created_at || ""}#${message.id || ""}`;
}
function updateActiveMessageLoadedAt(messages) {
    const latest = messages[messages.length - 1];
    activeMessageLoadedAt = getMessageMarker(latest);
}
function appendMessageIfNeeded(message) {
    if (!message || activeMessages.some((item) => item.id === message.id)) {
        return false;
    }
    activeMessages = [...activeMessages, message];
    updateActiveMessageLoadedAt(activeMessages);
    return true;
}
function markMessageRevoked(messageId) {
    const index = activeMessages.findIndex((item) => item.id === messageId);
    if (index === -1) {
        return false;
    }
    const target = activeMessages[index];
    const nextMessages = activeMessages.slice();
    nextMessages[index] = {
        ...target,
        deleted: true,
        content: target.content || "",
    };
    activeMessages = nextMessages;
    updateActiveMessageLoadedAt(activeMessages);
    return true;
}
function shouldRefreshActiveMessages(threadId) {
    const chat = chatCache.find((item) => item.id === threadId);
    if (!chat?.last_message_at) {
        return activeMessages.length === 0;
    }
    if (!activeMessageLoadedAt) {
        return true;
    }
    const [loadedAt] = activeMessageLoadedAt.split("#");
    return chat.last_message_at > loadedAt;
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
            <span>${truncatePreview(chat.last_message, 20)}</span>
            <span>${chat.last_message_at ? formatTime(chat.last_message_at) : ""}</span>
          </div>
          <div class="chat-item-meta">
            <span>${formatPresence(chat)}</span>
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
function buildChatListSignature(chats) {
    return chats
        .map((chat) => [
        chat.id,
        chat.other_username,
        chat.other_user_online ? "1" : "0",
        chat.other_user_device_type || "",
        chat.other_user_last_seen_at || "",
        chat.unread_count || 0,
        chat.last_message || "",
        chat.last_message_at || "",
    ].join("|"))
        .join("||");
}
async function loadChats(focusThreadId = null) {
    const { response, data } = await fetchChats();
    if (!response.ok) {
        chatList.innerHTML = "<div class='chat-empty'>无法加载会话</div>";
        return;
    }
    const nextChats = data.chats || [];
    const previousSignature = buildChatListSignature(chatCache);
    const nextSignature = buildChatListSignature(nextChats);
    chatCache = nextChats;
    if (focusThreadId) {
        const match = chatCache.find((item) => item.id === focusThreadId);
        if (match) {
            activeThreadId = match.id;
        }
    }
    if (previousSignature !== nextSignature) {
        renderChatList(chatCache);
    }
    updateActiveChatHeader();
}
async function startChatWithUser(userId) {
    const { response, data } = await startChat(userId);
    if (!response.ok) {
        chatSubtitle.textContent = data.error || "无法创建会话";
        return null;
    }
    return data.chat || null;
}
async function loadLLMThreads(threadId, preferredThreadId) {
    const chat = chatCache.find((item) => item.id === threadId);
    activeIsAIChat = isAIChat(chat);
    if (!activeIsAIChat) {
        activeLLMThreadId = null;
        activeLLMThreads = [];
        renderLLMThreadBar();
        return;
    }
    const { response, data } = await fetchLLMThreads(threadId, preferredThreadId || activeLLMThreadId);
    if (!response.ok) {
        activeLLMThreads = [];
        activeLLMThreadId = null;
        chatSubtitle.textContent = data.error || "无法加载话题";
        renderLLMThreadBar();
        return;
    }
    activeLLMThreads = data.threads || [];
    activeLLMThreadId = data.active_thread?.id || preferredThreadId || activeLLMThreads[0]?.id || null;
    renderLLMThreadBar();
}
function renderMessages(messages) {
    activeMessages = messages;
    updateActiveMessageLoadedAt(messages);
    if (!messages.length) {
        messageList.innerHTML = "<div class='chat-empty'>暂无消息</div>";
        return;
    }
    messageList.innerHTML = messages
        .map((msg) => {
        const isMine = msg.sender_id === currentUserId;
        const isSystem = msg.sender_id === "system";
        const isSharedMarkdown = msg.message_type === "shared_markdown" && Boolean(msg.markdown_entry_id);
        const isExpanded = expandedMarkdownMessages.has(String(msg.id));
        const expandedContent = sharedMarkdownContentCache.get(String(msg.id)) || "";
        const isLoadingExpanded = sharedMarkdownLoading.has(String(msg.id));
        const markdownActions = isSharedMarkdown
            ? `
            <div class="message-markdown-actions">
              <button class="btn-inline btn-secondary message-expand" data-id="${msg.id}" type="button">${isExpanded ? "缩小" : "放大"}</button>
              <button class="btn-inline btn-secondary message-copy" data-id="${msg.id}" type="button">复制</button>
              <button class="btn-inline btn-secondary message-public-share" data-id="${msg.id}" type="button">公开分享</button>
              <button class="btn-inline btn-secondary message-favorite" data-id="${msg.id}" type="button">收藏</button>
            </div>
          `
            : "";
        const content = msg.deleted
            ? "消息已撤回"
            : isSharedMarkdown
                ? `
              <div class="message-markdown-card">
                <div class="message-markdown-title">${escapeHtml(msg.markdown_title || "AI Markdown 回复")}</div>
                <div class="message-markdown-preview">${escapeHtml(msg.content || "")}</div>
                ${isExpanded ? `<div class="message-markdown-expanded markdown-body">${isLoadingExpanded ? "正在加载完整内容..." : renderMarkdown(expandedContent)}</div>` : ""}
                ${markdownActions}
              </div>
            `
                : isSystem
                    ? renderMarkdown(msg.content || "")
                    : escapeHtml(msg.content || "");
        const bubbleClass = msg.deleted ? "message-bubble deleted" : "message-bubble";
        const contentClass = isSharedMarkdown
            ? "message-bubble-content"
            : isSystem && !msg.deleted
                ? "message-bubble-content markdown-body"
                : "message-bubble-content";
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
            <div class="${bubbleClass}"><div class="${contentClass}">${content}</div></div>
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
    messageList.querySelectorAll(".message-copy").forEach((button) => {
        button.addEventListener("click", async () => {
            if (!activeThreadId) {
                return;
            }
            const messageId = button.dataset.id;
            if (!messageId) {
                return;
            }
            const { response, data } = await fetchSharedMarkdown(activeThreadId, messageId);
            if (!response.ok || !data.content) {
                return;
            }
            try {
                const copied = await copyTextToClipboard(data.content);
                if (!copied) {
                    chatSubtitle.textContent = "复制失败，请手动选择内容复制";
                    return;
                }
                chatSubtitle.textContent = "Markdown 已复制到剪贴板";
            }
            catch {
                chatSubtitle.textContent = "复制失败，请检查浏览器权限";
            }
        });
    });
    messageList.querySelectorAll(".message-expand").forEach((button) => {
        button.addEventListener("click", async () => {
            if (!activeThreadId) {
                return;
            }
            const messageId = button.dataset.id;
            if (!messageId) {
                return;
            }
            if (expandedMarkdownMessages.has(messageId)) {
                expandedMarkdownMessages.delete(messageId);
                renderMessages(activeMessages);
                return;
            }
            expandedMarkdownMessages.add(messageId);
            if (sharedMarkdownContentCache.has(messageId)) {
                renderMessages(activeMessages);
                return;
            }
            sharedMarkdownLoading.add(messageId);
            renderMessages(activeMessages);
            const { response, data } = await fetchSharedMarkdown(activeThreadId, messageId);
            sharedMarkdownLoading.delete(messageId);
            if (!response.ok || !data.content) {
                expandedMarkdownMessages.delete(messageId);
                chatSubtitle.textContent = data?.error || "无法加载完整 Markdown";
                renderMessages(activeMessages);
                return;
            }
            sharedMarkdownContentCache.set(messageId, data.content);
            renderMessages(activeMessages);
        });
    });
    messageList.querySelectorAll(".message-public-share").forEach((button) => {
        button.addEventListener("click", async () => {
            if (!activeThreadId) {
                return;
            }
            const messageId = button.dataset.id;
            if (!messageId) {
                return;
            }
            const { response, data } = await fetchSharedMarkdown(activeThreadId, messageId);
            if (!response.ok || !data.content) {
                return;
            }
            const shareResult = await requestJson("/api/markdown", {
                method: "POST",
                body: {
                    title: data.entry?.title || "AI Markdown 回复",
                    content: data.content,
                    is_public: true,
                },
            });
            if (!shareResult.response.ok || !shareResult.data.id) {
                chatSubtitle.textContent = shareResult.data.error || "公开分享失败";
                return;
            }
            chatSubtitle.textContent = "已公开分享";
            window.open(`/markdown.html?id=${encodeURIComponent(String(shareResult.data.id))}`, "_blank");
        });
    });
    messageList.querySelectorAll(".message-favorite").forEach((button) => {
        button.addEventListener("click", async () => {
            if (!activeThreadId) {
                return;
            }
            const messageId = button.dataset.id;
            if (!messageId) {
                return;
            }
            const { response, data } = await fetchSharedMarkdown(activeThreadId, messageId);
            if (!response.ok || !data.content) {
                return;
            }
            const saveResult = await requestJson("/api/markdown", {
                method: "POST",
                body: {
                    title: data.entry?.title || "AI Markdown 回复",
                    content: data.content,
                    is_public: false,
                },
            });
            if (!saveResult.response.ok || !saveResult.data.id) {
                chatSubtitle.textContent = saveResult.data.error || "收藏失败";
                return;
            }
            chatSubtitle.textContent = "已收藏到我的 Markdown";
            window.open(`/editor.html?id=${encodeURIComponent(String(saveResult.data.id))}`, "_blank");
        });
    });
    messageList.scrollTop = messageList.scrollHeight;
}
async function loadMessages(threadId) {
    messageList.innerHTML = "<div class='chat-empty'>加载中...</div>";
    const { response, data } = await fetchMessages(threadId, 200, activeLLMThreadId);
    if (!response.ok) {
        messageList.innerHTML = "<div class='chat-empty'>无法加载消息</div>";
        return;
    }
    if (data.active_thread?.id) {
        activeLLMThreadId = data.active_thread.id;
        if (!activeLLMThreads.some((thread) => thread.id === data.active_thread?.id) && data.active_thread) {
            activeLLMThreads = [data.active_thread, ...activeLLMThreads];
        }
        renderLLMThreadBar();
    }
    renderMessages(data.messages || []);
}
async function refreshActiveMessagesIfNeeded(threadId, force = false) {
    if (force || shouldRefreshActiveMessages(threadId)) {
        await loadMessages(threadId);
    }
}
async function openChat(chat) {
    activeThreadId = chat.id;
    activeMessageLoadedAt = "";
    activeLLMThreadId = null;
    updateActiveChatHeader();
    messageInput.disabled = false;
    renderChatList(chatCache);
    await loadLLMThreads(chat.id);
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
    if (!markMessageRevoked(messageId)) {
        await loadMessages(activeThreadId);
    }
    else {
        renderMessages(activeMessages);
    }
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
            await refreshActiveMessagesIfNeeded(activeThreadId);
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
            if (activeThreadId === chatId && chatId && payload.message) {
                const incomingLLMThreadId = payload.message.llm_thread_id || null;
                if ((!activeLLMThreadId && !incomingLLMThreadId) || activeLLMThreadId === incomingLLMThreadId) {
                    if (appendMessageIfNeeded(payload.message)) {
                        renderMessages(activeMessages);
                    }
                }
                else if (activeIsAIChat) {
                    await loadLLMThreads(chatId, activeLLMThreadId);
                }
                if (payload.message.llm_thread_id && !activeLLMThreads.some((thread) => thread.id === payload.message?.llm_thread_id)) {
                    await loadLLMThreads(chatId, activeLLMThreadId);
                }
            }
            await loadChats(activeThreadId);
            return;
        }
        if (payload.type === "read") {
            await loadChats(activeThreadId);
            return;
        }
        if (payload.type === "presence") {
            await loadChats(activeThreadId);
            return;
        }
        if (payload.type === "revoke") {
            if (activeThreadId === chatId && chatId && payload.message_id) {
                if (markMessageRevoked(payload.message_id)) {
                    renderMessages(activeMessages);
                }
                else {
                    await loadMessages(chatId);
                }
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
    const response = await sendMessage(activeThreadId, content, activeLLMThreadId);
    if (!response.ok) {
        return;
    }
    messageInput.value = "";
    if (!wsConnected) {
        await loadMessages(activeThreadId);
    }
    await loadChats(activeThreadId);
});
chatRefreshBtn.addEventListener("click", async () => {
    await loadChats(activeThreadId);
    if (activeThreadId) {
        await loadLLMThreads(activeThreadId, activeLLMThreadId);
        await refreshActiveMessagesIfNeeded(activeThreadId, true);
    }
});
chatThreadSelect.addEventListener("change", async () => {
    if (!activeThreadId) {
        return;
    }
    const nextID = Number(chatThreadSelect.value || 0);
    activeLLMThreadId = nextID > 0 ? nextID : null;
    activeMessageLoadedAt = "";
    await loadMessages(activeThreadId);
});
chatNewTopicBtn.addEventListener("click", async () => {
    if (!activeThreadId) {
        return;
    }
    const result = await createLLMThread(activeThreadId, "");
    if (!result.response.ok) {
        chatSubtitle.textContent = result.data.error || "创建新话题失败";
        return;
    }
    activeLLMThreads = result.data.threads || activeLLMThreads;
    activeLLMThreadId = result.data.thread?.id || activeLLMThreads[0]?.id || null;
    activeMessages = [];
    activeMessageLoadedAt = "";
    renderLLMThreadBar();
    renderMessages([]);
    if (activeThreadId) {
        await loadMessages(activeThreadId);
    }
});
async function init() {
    await hydrateSiteBrand();
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
