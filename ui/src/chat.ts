import { createLLMThread, fetchChatLLMConfigs, fetchChats, fetchLLMThreads, fetchMessages, fetchSharedMarkdown, retryMessage, revokeMessage as revokeChatMessage, sendMessage, startChat, switchLLMThreadConfig, updateLLMThread } from "./api/chat.js";
import { requestJson } from "./api/http.js";
import { fetchCurrentUser } from "./api/session.js";
import { resolveAvatar } from "./lib/avatar.js";
import { formatDeviceType } from "./lib/client.js";
import { byId } from "./lib/dom.js";
import { renderMarkdown } from "./lib/marked.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";
import type { ChatEventPayload, ChatLLMConfig, ChatMessage, ChatSummary, LLMThread } from "./types/chat.js";
const chatWelcome = byId<HTMLElement>("chatWelcome");
const chatList = byId<HTMLElement>("chatList");
const chatTitle = byId<HTMLElement>("chatTitle");
const chatSubtitle = byId<HTMLElement>("chatSubtitle");
const messageList = byId<HTMLElement>("messageList");
const messageForm = byId<HTMLFormElement>("messageForm");
const messageInput = byId<HTMLInputElement>("messageInput");
const chatRefreshBtn = byId<HTMLButtonElement>("chatRefreshBtn");
const chatNewTopicBtn = byId<HTMLButtonElement>("chatNewTopicBtn");
const chatRenameTopicBtn = byId<HTMLButtonElement>("chatRenameTopicBtn");
const chatThreadBar = byId<HTMLElement>("chatThreadBar");
const chatThreadSelect = byId<HTMLSelectElement>("chatThreadSelect");
const chatModelBar = byId<HTMLElement>("chatModelBar");
const chatModelCurrent = byId<HTMLElement>("chatModelCurrent");
const chatModelSelect = byId<HTMLSelectElement>("chatModelSelect");
const chatSwitchModelBtn = byId<HTMLButtonElement>("chatSwitchModelBtn");

let currentUserId = "";
let activeThreadId: string | null = null;
let chatCache: ChatSummary[] = [];
let pollTimer: number | null = null;
let ws: WebSocket | null = null;
let wsConnected = false;
let activeMessages: ChatMessage[] = [];
let activeMessageLoadedAt = "";
let activeLLMThreadId: number | null = null;
let activeLLMThreads: LLMThread[] = [];
let activeIsAIChat = false;
let activeIsBotChat = false;
let currentLLMConfigs: ChatLLMConfig[] = [];
const expandedMarkdownMessages = new Set<string>();
const sharedMarkdownContentCache = new Map<string, string>();
const sharedMarkdownLoading = new Set<string>();

function escapeHtml(input: string): string {
  return input
    .split("&").join("&amp;")
    .split("<").join("&lt;")
    .split(">").join("&gt;")
    .split('"').join("&quot;")
    .split("'").join("&#39;");
}

async function copyTextToClipboard(text: string): Promise<boolean> {
  if (!text) {
    return false;
  }

  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
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
  } catch {
    copied = false;
  } finally {
    document.body.removeChild(textarea);
  }

  return copied;
}

initStoredTheme();
bindThemeSync();

function formatTime(value?: string): string {
  if (!value) {
    return "";
  }
  return new Date(value).toLocaleString();
}

function formatPresence(chat: ChatSummary): string {
  if (chat.other_user_online) {
    return `在线 · ${formatDeviceType(chat.other_user_device_type)}`;
  }
  if (chat.other_user_last_seen_at) {
    return `离线 · 上次在线 ${formatTime(chat.other_user_last_seen_at)}`;
  }
  return `离线 · 最近设备 ${formatDeviceType(chat.other_user_device_type)}`;
}

function truncatePreview(input?: string, maxLength = 20): string {
  const text = (input || "").trim();
  if (!text) {
    return "暂无消息";
  }
  return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text;
}

function updateActiveChatHeader(): void {
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

function isAIChat(chat?: ChatSummary | null): boolean {
  if (!chat) {
    return false;
  }
  return chat.other_user_id === "system" || chat.other_user_id.startsWith("bot_");
}

function isBotChat(chat?: ChatSummary | null): boolean {
  if (!chat) {
    return false;
  }
  return chat.other_user_id.startsWith("bot_");
}

function renderLLMThreadBar(): void {
  chatThreadBar.hidden = !activeIsAIChat;
  chatNewTopicBtn.hidden = !activeIsAIChat;
  chatRenameTopicBtn.hidden = !activeIsAIChat;
  if (!activeIsAIChat) {
    chatThreadSelect.innerHTML = "";
    chatModelBar.hidden = true;
    return;
  }
  if (!activeLLMThreads.length) {
    chatThreadSelect.innerHTML = `<option value="">默认话题</option>`;
    chatThreadSelect.disabled = true;
    chatRenameTopicBtn.disabled = true;
    renderLLMThreadModelBar();
    return;
  }
  chatThreadSelect.disabled = false;
  chatRenameTopicBtn.disabled = !activeLLMThreadId;
  chatThreadSelect.innerHTML = activeLLMThreads
    .map((thread) => `<option value="${thread.id}">${escapeHtml(thread.title || "新话题")}</option>`)
    .join("");
  if (activeLLMThreadId) {
    chatThreadSelect.value = String(activeLLMThreadId);
  }
  renderLLMThreadModelBar();
}

function renderLLMThreadModelBar(): void {
  chatModelBar.hidden = !activeIsBotChat;
  if (!activeIsBotChat) {
    chatModelCurrent.textContent = "官方 system 助理";
    chatModelSelect.innerHTML = "";
    chatSwitchModelBtn.disabled = true;
    return;
  }
  const activeThread = activeLLMThreads.find((thread) => thread.id === activeLLMThreadId) || null;
  if (!activeThread) {
    chatModelCurrent.textContent = "未选择话题";
    chatModelSelect.innerHTML = `<option value="">请先选择话题</option>`;
    chatSwitchModelBtn.disabled = true;
    return;
  }
  chatModelCurrent.textContent = activeThread.config_name
    ? `${activeThread.config_name}${activeThread.config_model ? ` · ${activeThread.config_model}` : ""}`
    : "跟随 Bot 默认配置";
  if (!currentLLMConfigs.length) {
    chatModelSelect.innerHTML = `<option value="">暂无可用配置</option>`;
    chatSwitchModelBtn.disabled = true;
    return;
  }
  chatModelSelect.innerHTML = currentLLMConfigs
    .map((config) => `<option value="${config.id}">${escapeHtml(config.name)} · ${escapeHtml(config.model)}</option>`)
    .join("");
  if (activeThread.llm_config_id && currentLLMConfigs.some((config) => config.id === activeThread.llm_config_id)) {
    chatModelSelect.value = String(activeThread.llm_config_id);
  } else if (currentLLMConfigs[0]) {
    chatModelSelect.value = String(currentLLMConfigs[0].id);
  }
  chatSwitchModelBtn.disabled = activeThread.llm_config_id != null && chatModelSelect.value === String(activeThread.llm_config_id);
}

function getMessageMarker(message?: ChatMessage | null): string {
  if (!message) {
    return "";
  }
  return `${message.created_at || ""}#${message.id || ""}`;
}

function updateActiveMessageLoadedAt(messages: ChatMessage[]): void {
  const latest = messages[messages.length - 1];
  activeMessageLoadedAt = getMessageMarker(latest);
}

function appendMessageIfNeeded(message: ChatMessage): boolean {
  if (!message || activeMessages.some((item) => item.id === message.id)) {
    return false;
  }
  activeMessages = [...activeMessages, message];
  updateActiveMessageLoadedAt(activeMessages);
  return true;
}

function markMessageRevoked(messageId: string): boolean {
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

function removeMessageIfNeeded(messageId: string): boolean {
  const nextMessages = activeMessages.filter((item) => item.id !== messageId);
  if (nextMessages.length === activeMessages.length) {
    return false;
  }
  activeMessages = nextMessages;
  updateActiveMessageLoadedAt(activeMessages);
  return true;
}

function shouldRefreshActiveMessages(threadId: string): boolean {
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

function buildChatListSignature(chats: ChatSummary[]): string {
  return chats
    .map((chat) =>
      [
        chat.id,
        chat.other_username,
        chat.other_user_online ? "1" : "0",
        chat.other_user_device_type || "",
        chat.other_user_last_seen_at || "",
        chat.unread_count || 0,
        chat.last_message || "",
        chat.last_message_at || "",
      ].join("|")
    )
    .join("||");
}

async function loadChats(focusThreadId: string | null = null): Promise<void> {
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

async function startChatWithUser(userId: string): Promise<ChatSummary | null> {
  const { response, data } = await startChat(userId);
  if (!response.ok) {
    chatSubtitle.textContent = data.error || "无法创建会话";
    return null;
  }
  return data.chat || null;
}

async function loadLLMThreads(threadId: string, preferredThreadId?: number | null): Promise<void> {
  const chat = chatCache.find((item) => item.id === threadId);
  activeIsAIChat = isAIChat(chat);
  activeIsBotChat = isBotChat(chat);
  if (!activeIsAIChat) {
    activeLLMThreadId = null;
    activeLLMThreads = [];
    currentLLMConfigs = [];
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

async function loadChatLLMConfigs(): Promise<void> {
  if (!activeIsBotChat) {
    currentLLMConfigs = [];
    renderLLMThreadModelBar();
    return;
  }
  const { response, data } = await fetchChatLLMConfigs();
  if (!response.ok) {
    currentLLMConfigs = [];
    chatSubtitle.textContent = data.error || "无法加载模型配置";
    renderLLMThreadModelBar();
    return;
  }
  currentLLMConfigs = data.configs || [];
  renderLLMThreadModelBar();
}

function renderMessages(messages: ChatMessage[]): void {
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
      const isBotReply = !isMine && (msg.sender_id === "system" || msg.sender_id.startsWith("bot_"));
      const isFailedBotReply = isBotReply && Boolean(msg.failed) && !msg.deleted;
      const isSharedMarkdown = msg.message_type === "shared_markdown" && Boolean(msg.markdown_entry_id);
      const isExpanded = expandedMarkdownMessages.has(String(msg.id));
      const expandedContent = sharedMarkdownContentCache.get(String(msg.id)) || "";
      const isLoadingExpanded = sharedMarkdownLoading.has(String(msg.id));
      const retryAction = isFailedBotReply
        ? `<button class="btn-inline btn-secondary message-retry" data-id="${msg.id}" type="button">Retry</button>`
        : "";
      const failureBadge = isFailedBotReply
        ? `<div class="message-failure-badge">发送失败，可重试</div>`
        : "";
      const markdownActions = isSharedMarkdown
        ? `
            <div class="message-markdown-actions">
              <button class="btn-inline btn-secondary message-expand" data-id="${msg.id}" type="button">${isExpanded ? "缩小" : "放大"}</button>
              ${retryAction}
              <button class="btn-inline btn-secondary message-copy" data-id="${msg.id}" type="button">复制</button>
              <button class="btn-inline btn-secondary message-public-share" data-id="${msg.id}" type="button">公开分享</button>
              <button class="btn-inline btn-secondary message-favorite" data-id="${msg.id}" type="button">收藏</button>
            </div>
          `
        : "";
      const textActions = !isSharedMarkdown && retryAction
        ? `<div class="message-inline-actions">${retryAction}</div>`
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
      const revokeButton =
        isMine && !msg.deleted
          ? `<button class="message-revoke" data-id="${msg.id}" type="button">撤回</button>`
          : "";
      const avatar = resolveAvatar(msg.sender_username, msg.sender_icon, 48);
      const activeThread = activeLLMThreads.find((thread) => thread.id === (msg.llm_thread_id || activeLLMThreadId)) || null;
      const botModelMeta = isBotReply
        ? (activeThread?.config_model || activeThread?.config_name || "LLM")
        : "";
      const messageMeta = isBotReply
        ? `${msg.sender_username} · ${botModelMeta} · ${formatTime(msg.created_at)}${isFailedBotReply ? " · 失败" : ""}`
        : `${msg.sender_username} · ${formatTime(msg.created_at)}`;
      return `
        <div class="message-item ${isMine ? "mine" : "other"}">
          <div class="message-head">
            <img class="avatar-xs" src="${avatar}" alt="${msg.sender_username}" />
            <div class="message-meta">${messageMeta}</div>
          </div>
          <div class="message-row">
            <div class="${bubbleClass}"><div class="${contentClass}">${content}</div>${failureBadge}${textActions}</div>
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
  messageList.querySelectorAll<HTMLButtonElement>(".message-retry").forEach((button) => {
    button.addEventListener("click", async () => {
      if (!activeThreadId) {
        return;
      }
      const messageId = button.dataset.id;
      if (!messageId) {
        return;
      }
      button.disabled = true;
      chatSubtitle.textContent = "正在重试上一条用户消息...";
      try {
        const { response, data } = await retryMessage(activeThreadId, messageId);
        if (!response.ok) {
          chatSubtitle.textContent = data.error || "重试失败";
          return;
        }
        if (removeMessageIfNeeded(messageId)) {
          renderMessages(activeMessages);
        }
        chatSubtitle.textContent = data.message || "已重新提交上一条用户消息";
      } finally {
        button.disabled = false;
      }
    });
  });
  messageList.querySelectorAll<HTMLButtonElement>(".message-copy").forEach((button) => {
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
      } catch {
        chatSubtitle.textContent = "复制失败，请检查浏览器权限";
      }
    });
  });
  messageList.querySelectorAll<HTMLButtonElement>(".message-expand").forEach((button) => {
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
  messageList.querySelectorAll<HTMLButtonElement>(".message-public-share").forEach((button) => {
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
      const shareResult = await requestJson<{ id?: number; error?: string }>("/api/markdown", {
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
  messageList.querySelectorAll<HTMLButtonElement>(".message-favorite").forEach((button) => {
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
      const saveResult = await requestJson<{ id?: number; error?: string }>("/api/markdown", {
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

async function loadMessages(threadId: string): Promise<void> {
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

async function refreshActiveMessagesIfNeeded(threadId: string, force = false): Promise<void> {
  if (force || shouldRefreshActiveMessages(threadId)) {
    await loadMessages(threadId);
  }
}

async function openChat(chat: ChatSummary): Promise<void> {
  activeThreadId = chat.id;
  activeMessageLoadedAt = "";
  activeLLMThreadId = null;
  updateActiveChatHeader();
  messageInput.disabled = false;
  renderChatList(chatCache);
  await loadLLMThreads(chat.id);
  await loadChatLLMConfigs();
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
  if (!markMessageRevoked(messageId)) {
    await loadMessages(activeThreadId);
  } else {
    renderMessages(activeMessages);
  }
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
      await refreshActiveMessagesIfNeeded(activeThreadId);
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
      if (activeThreadId === chatId && chatId && payload.message) {
        const incomingLLMThreadId = payload.message.llm_thread_id || null;
        if ((!activeLLMThreadId && !incomingLLMThreadId) || activeLLMThreadId === incomingLLMThreadId) {
          if (appendMessageIfNeeded(payload.message)) {
            renderMessages(activeMessages);
          }
        } else if (activeIsAIChat) {
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
        const handled = payload.user_id === "retry"
          ? removeMessageIfNeeded(payload.message_id)
          : markMessageRevoked(payload.message_id);
        if (handled) {
          renderMessages(activeMessages);
        } else {
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
  renderLLMThreadModelBar();
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

chatRenameTopicBtn.addEventListener("click", async () => {
  if (!activeThreadId || !activeLLMThreadId) {
    return;
  }
  const currentThread = activeLLMThreads.find((thread) => thread.id === activeLLMThreadId);
  const nextTitle = window.prompt("输入新的话题标题", currentThread?.title || "新话题");
  if (nextTitle == null) {
    return;
  }
  const trimmed = nextTitle.trim();
  if (!trimmed) {
    chatSubtitle.textContent = "话题标题不能为空";
    return;
  }
  const result = await updateLLMThread(activeThreadId, activeLLMThreadId, trimmed);
  if (!result.response.ok) {
    chatSubtitle.textContent = result.data.error || "重命名失败";
    return;
  }
  activeLLMThreads = result.data.threads || activeLLMThreads;
  renderLLMThreadBar();
  chatSubtitle.textContent = result.data.message || "话题标题已更新";
});

chatModelSelect.addEventListener("change", () => {
  const currentThread = activeLLMThreads.find((thread) => thread.id === activeLLMThreadId) || null;
  chatSwitchModelBtn.disabled = !currentThread || (currentThread.llm_config_id != null && chatModelSelect.value === String(currentThread.llm_config_id));
});

chatSwitchModelBtn.addEventListener("click", async () => {
  if (!activeThreadId || !activeLLMThreadId) {
    return;
  }
  const llmConfigId = Number(chatModelSelect.value || 0);
  if (llmConfigId <= 0) {
    chatSubtitle.textContent = "请选择要切换的模型配置";
    return;
  }
  const result = await switchLLMThreadConfig(activeThreadId, activeLLMThreadId, llmConfigId);
  if (!result.response.ok) {
    chatSubtitle.textContent = result.data.error || "切换模型失败";
    return;
  }
  activeLLMThreads = result.data.threads || activeLLMThreads;
  renderLLMThreadBar();
  chatSubtitle.textContent = result.data.message || "当前话题模型已切换";
});

async function init(): Promise<void> {
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
