import { createLLMThread, fetchChatLLMConfigs, fetchChats, fetchLLMThreads, fetchMessages, fetchSharedMarkdown, retryMessage, revokeMessage as revokeChatMessage, sendAttachment, sendMessage, startChat, switchLLMThreadConfig, updateLLMThread } from "./api/chat.js";
import { requestJson } from "./api/http.js";
import { fetchCurrentUser } from "./api/session.js";
import { resolveAvatar } from "./lib/avatar.js";
import { formatDeviceType } from "./lib/client.js";
import { byId } from "./lib/dom.js";
import { renderMarkdown } from "./lib/marked.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";
import { t } from "./lib/i18n.js";
const chatWelcome = byId("chatWelcome");
const chatList = byId("chatList");
const chatTitle = byId("chatTitle");
const chatSubtitle = byId("chatSubtitle");
const messageList = byId("messageList");
const messageForm = byId("messageForm");
const messageInput = byId("messageInput");
const chatRefreshBtn = byId("chatRefreshBtn");
const chatNewTopicBtn = byId("chatNewTopicBtn");
const chatRenameTopicBtn = byId("chatRenameTopicBtn");
const chatThreadBar = byId("chatThreadBar");
const chatThreadSelect = byId("chatThreadSelect");
const chatModelBar = byId("chatModelBar");
const chatModelCurrent = byId("chatModelCurrent");
const chatModelSelect = byId("chatModelSelect");
const chatSwitchModelBtn = byId("chatSwitchModelBtn");
const attachmentBtn = byId("attachmentBtn");
const attachmentInput = byId("attachmentInput");
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
let activeIsBotChat = false;
let currentLLMConfigs = [];
let activeChatSummary = null;
let activeChatBlocked = false;
let activeChatBlockMessage = "";
let activeChatReplyRequired = false;
let activeChatReplyRequiredMessage = "";
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
function formatFileSize(bytes) {
    if (bytes >= 1024 * 1024) {
        return t("chat.fileSizeMB", { size: (bytes / (1024 * 1024)).toFixed(1) });
    }
    return t("chat.fileSizeKB", { size: String(Math.ceil(bytes / 1024)) });
}
function attachmentIcon(mimeType) {
    if (mimeType.startsWith("image/"))
        return "🖼️";
    if (mimeType.startsWith("video/"))
        return "🎬";
    if (mimeType.startsWith("audio/"))
        return "🎵";
    if (mimeType.includes("pdf"))
        return "📄";
    if (mimeType.includes("zip") || mimeType.includes("compress"))
        return "🗜️";
    return "📎";
}
function renderAttachment(att) {
    const isImage = att.mime_type.startsWith("image/");
    const isVideo = att.mime_type.startsWith("video/");
    const fileUrl = escapeHtml(att.url);
    const fileName = escapeHtml(att.file_name);
    const sizeStr = escapeHtml(formatFileSize(att.size));
    if (isImage) {
        const thumbUrl = escapeHtml(att.thumbnail_url || att.url);
        return `<a href="${fileUrl}" target="_blank" rel="noopener"><img class="message-attachment-image" src="${thumbUrl}" alt="${fileName}" title="${fileName}" /></a>`;
    }
    if (isVideo) {
        return `<video class="message-attachment-image" src="${fileUrl}" controls preload="metadata"></video>`;
    }
    const icon = attachmentIcon(att.mime_type);
    return `
    <a class="message-attachment-file" href="${fileUrl}" target="_blank" rel="noopener" download="${fileName}">
      <span class="message-attachment-icon">${icon}</span>
      <div class="message-attachment-meta">
        <div class="message-attachment-name">${fileName}</div>
        <div class="message-attachment-size">${sizeStr}</div>
      </div>
    </a>
  `;
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
        return t("chat.online", { device: formatDeviceType(chat.other_user_device_type, t) });
    }
    if (chat.other_user_last_seen_at) {
        return t("chat.offline", { time: formatTime(chat.other_user_last_seen_at) });
    }
    return t("chat.offlineDevice", { device: formatDeviceType(chat.other_user_device_type, t) });
}
function truncatePreview(input, maxLength = 20) {
    const text = (input || "").trim();
    if (!text) {
        return t("chat.noPreview");
    }
    return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text;
}
function updateActiveChatHeader() {
    if (!activeThreadId && !activeChatSummary) {
        return;
    }
    const chat = chatCache.find((item) => item.id === activeThreadId) || activeChatSummary;
    if (!chat) {
        return;
    }
    activeChatSummary = chat;
    chatTitle.textContent = chat.other_username;
    chatSubtitle.textContent = activeChatBlocked && activeChatBlockMessage
        ? activeChatBlockMessage
        : activeChatReplyRequired && activeChatReplyRequiredMessage
            ? activeChatReplyRequiredMessage
            : formatPresence(chat);
}
function isAIChat(chat) {
    if (!chat) {
        return false;
    }
    return chat.other_user_id === "system" || chat.other_user_id.startsWith("bot_");
}
function isBotChat(chat) {
    if (!chat) {
        return false;
    }
    return chat.other_user_id.startsWith("bot_");
}
function renderLLMThreadBar() {
    chatThreadBar.hidden = !activeIsAIChat;
    chatNewTopicBtn.hidden = !activeIsAIChat;
    chatRenameTopicBtn.hidden = !activeIsAIChat;
    if (!activeIsAIChat) {
        chatThreadSelect.innerHTML = "";
        chatModelBar.hidden = true;
        return;
    }
    if (!activeLLMThreads.length) {
        chatThreadSelect.innerHTML = `<option value="">${t("chat.defaultTopic")}</option>`;
        chatThreadSelect.disabled = true;
        chatRenameTopicBtn.disabled = true;
        renderLLMThreadModelBar();
        return;
    }
    chatThreadSelect.disabled = false;
    chatRenameTopicBtn.disabled = !activeLLMThreadId;
    chatThreadSelect.innerHTML = activeLLMThreads
        .map((thread) => `<option value="${thread.id}">${escapeHtml(thread.title || t("chat.newTopic"))}</option>`)
        .join("");
    if (activeLLMThreadId) {
        chatThreadSelect.value = String(activeLLMThreadId);
    }
    renderLLMThreadModelBar();
}
function renderLLMThreadModelBar() {
    chatModelBar.hidden = !activeIsBotChat;
    if (!activeIsBotChat) {
        chatModelCurrent.textContent = t("chat.systemAssistant");
        chatModelSelect.innerHTML = "";
        chatSwitchModelBtn.disabled = true;
        return;
    }
    const activeThread = activeLLMThreads.find((thread) => thread.id === activeLLMThreadId) || null;
    if (!activeThread) {
        chatModelCurrent.textContent = t("chat.noTopicSelected");
        chatModelSelect.innerHTML = `<option value="">${t("chat.selectTopicFirst")}</option>`;
        chatSwitchModelBtn.disabled = true;
        return;
    }
    chatModelCurrent.textContent = activeThread.config_name
        ? `${activeThread.config_name}${activeThread.config_model ? ` · ${activeThread.config_model}` : ""}`
        : t("chat.followBotDefault");
    if (!currentLLMConfigs.length) {
        chatModelSelect.innerHTML = `<option value="">${t("chat.noConfigs")}</option>`;
        chatSwitchModelBtn.disabled = true;
        return;
    }
    chatModelSelect.innerHTML = currentLLMConfigs
        .map((config) => `<option value="${config.id}">${escapeHtml(config.name)} · ${escapeHtml(config.model)}</option>`)
        .join("");
    if (activeThread.llm_config_id && currentLLMConfigs.some((config) => config.id === activeThread.llm_config_id)) {
        chatModelSelect.value = String(activeThread.llm_config_id);
    }
    else if (currentLLMConfigs[0]) {
        chatModelSelect.value = String(currentLLMConfigs[0].id);
    }
    chatSwitchModelBtn.disabled = activeThread.llm_config_id != null && chatModelSelect.value === String(activeThread.llm_config_id);
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
function removeMessageIfNeeded(messageId) {
    const nextMessages = activeMessages.filter((item) => item.id !== messageId);
    if (nextMessages.length === activeMessages.length) {
        return false;
    }
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
    chatWelcome.textContent = t("chat.welcome", { username: data.username });
}
function renderChatList(chats) {
    chatList.innerHTML = "";
    if (!chats.length) {
        chatList.innerHTML = `<div class='chat-empty'>${t("chat.noConversations")}</div>`;
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
        chatList.innerHTML = `<div class='chat-empty'>${t("chat.loadFailed")}</div>`;
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
            activeChatSummary = match;
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
        chatSubtitle.textContent = data.error || t("chat.createFailed");
        return null;
    }
    return data.chat || null;
}
async function loadLLMThreads(threadId, preferredThreadId) {
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
        chatSubtitle.textContent = data.error || t("chat.topicsLoadFailed");
        renderLLMThreadBar();
        return;
    }
    activeLLMThreads = data.threads || [];
    activeLLMThreadId = data.active_thread?.id || preferredThreadId || activeLLMThreads[0]?.id || null;
    renderLLMThreadBar();
}
async function loadChatLLMConfigs() {
    if (!activeIsBotChat) {
        currentLLMConfigs = [];
        renderLLMThreadModelBar();
        return;
    }
    const { response, data } = await fetchChatLLMConfigs();
    if (!response.ok) {
        currentLLMConfigs = [];
        chatSubtitle.textContent = data.error || t("chat.modelConfigLoadFailed");
        renderLLMThreadModelBar();
        return;
    }
    currentLLMConfigs = data.configs || [];
    renderLLMThreadModelBar();
}
function renderMessages(messages) {
    activeMessages = messages;
    updateActiveMessageLoadedAt(messages);
    if (!messages.length) {
        messageList.innerHTML = `<div class='chat-empty'>${t("chat.noMessages")}</div>`;
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
            ? `<div class="message-failure-badge">${t("chat.sendFailed")}</div>`
            : "";
        const markdownActions = isSharedMarkdown
            ? `
            <div class="message-markdown-actions">
              <button class="btn-inline btn-secondary message-expand" data-id="${msg.id}" type="button">${isExpanded ? t("chat.collapse") : t("chat.expand")}</button>
              ${retryAction}
              <button class="btn-inline btn-secondary message-copy" data-id="${msg.id}" type="button">${t("chat.copy")}</button>
              <button class="btn-inline btn-secondary message-public-share" data-id="${msg.id}" type="button">${t("chat.sharePublicly")}</button>
              <button class="btn-inline btn-secondary message-favorite" data-id="${msg.id}" type="button">${t("chat.favorite")}</button>
            </div>
          `
            : "";
        const textActions = !isSharedMarkdown && retryAction
            ? `<div class="message-inline-actions">${retryAction}</div>`
            : "";
        const isAttachment = msg.message_type === "attachment" && Boolean(msg.attachment);
        const content = msg.deleted
            ? t("chat.messageRevoked")
            : isAttachment && msg.attachment
                ? renderAttachment(msg.attachment)
                : isSharedMarkdown
                    ? `
              <div class="message-markdown-card">
                <div class="message-markdown-title">${escapeHtml(msg.markdown_title || t("chat.aiMarkdownReply"))}</div>
                <div class="message-markdown-preview">${escapeHtml(msg.content || "")}</div>
                ${isExpanded ? `<div class="message-markdown-expanded markdown-body">${isLoadingExpanded ? t("chat.loadingContent") : renderMarkdown(expandedContent)}</div>` : ""}
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
            ? `<button class="message-revoke" data-id="${msg.id}" type="button">${t("chat.revoke")}</button>`
            : "";
        const avatar = resolveAvatar(msg.sender_username, msg.sender_icon, 48);
        const activeThread = activeLLMThreads.find((thread) => thread.id === (msg.llm_thread_id || activeLLMThreadId)) || null;
        const botModelMeta = isBotReply
            ? (activeThread?.config_model || activeThread?.config_name || "LLM")
            : "";
        const messageMeta = isBotReply
            ? `${msg.sender_username} · ${botModelMeta} · ${formatTime(msg.created_at)}${isFailedBotReply ? ` · ${t("chat.failed")}` : ""}`
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
    messageList.querySelectorAll(".message-revoke").forEach((button) => {
        button.addEventListener("click", async () => {
            const messageId = button.dataset.id;
            if (!messageId) {
                return;
            }
            await revokeMessage(messageId);
        });
    });
    messageList.querySelectorAll(".message-retry").forEach((button) => {
        button.addEventListener("click", async () => {
            if (!activeThreadId) {
                return;
            }
            const messageId = button.dataset.id;
            if (!messageId) {
                return;
            }
            button.disabled = true;
            chatSubtitle.textContent = t("chat.retrying");
            try {
                const { response, data } = await retryMessage(activeThreadId, messageId);
                if (!response.ok) {
                    chatSubtitle.textContent = data.error || t("chat.retryFailed");
                    return;
                }
                if (removeMessageIfNeeded(messageId)) {
                    renderMessages(activeMessages);
                }
                chatSubtitle.textContent = data.message || t("chat.retrySuccess");
            }
            finally {
                button.disabled = false;
            }
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
                    chatSubtitle.textContent = t("chat.copyFailed");
                    return;
                }
                chatSubtitle.textContent = t("chat.copySuccess");
            }
            catch {
                chatSubtitle.textContent = t("chat.copyPermissionFailed");
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
                chatSubtitle.textContent = data?.error || t("chat.markdownLoadFailed");
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
                    title: data.entry?.title || t("chat.aiMarkdownReply"),
                    content: data.content,
                    is_public: true,
                },
            });
            if (!shareResult.response.ok || !shareResult.data.id) {
                chatSubtitle.textContent = shareResult.data.error || t("chat.shareFailed");
                return;
            }
            chatSubtitle.textContent = t("chat.shareSuccess");
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
                    title: data.entry?.title || t("chat.aiMarkdownReply"),
                    content: data.content,
                    is_public: false,
                },
            });
            if (!saveResult.response.ok || !saveResult.data.id) {
                chatSubtitle.textContent = saveResult.data.error || t("chat.favoriteFailed");
                return;
            }
            chatSubtitle.textContent = t("chat.favoriteSuccess");
            window.open(`/editor.html?id=${encodeURIComponent(String(saveResult.data.id))}`, "_blank");
        });
    });
    messageList.scrollTop = messageList.scrollHeight;
}
async function loadMessages(threadId) {
    messageList.innerHTML = `<div class='chat-empty'>${t("chat.loading")}</div>`;
    const { response, data } = await fetchMessages(threadId, 200, activeLLMThreadId);
    if (!response.ok) {
        messageList.innerHTML = `<div class='chat-empty'>${t("chat.loadFailed")}</div>`;
        return;
    }
    activeChatBlocked = Boolean(data.blocked);
    activeChatBlockMessage = data.block_message || "";
    activeChatReplyRequired = Boolean(data.reply_required);
    activeChatReplyRequiredMessage = data.reply_required_message || "";
    const inputDisabled = activeChatBlocked || activeChatReplyRequired;
    messageInput.disabled = inputDisabled;
    attachmentBtn.classList.toggle("uploading", inputDisabled);
    updateActiveChatHeader();
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
    const previousThreadId = activeThreadId;
    activeThreadId = chat.id;
    activeChatSummary = chat;
    activeChatBlocked = false;
    activeChatBlockMessage = "";
    activeChatReplyRequired = false;
    activeChatReplyRequiredMessage = "";
    activeMessageLoadedAt = "";
    activeLLMThreadId = null;
    updateActiveChatHeader();
    messageInput.disabled = false;
    renderChatList(chatCache);
    if (previousThreadId && previousThreadId !== chat.id) {
        sendPresence("leave_thread", previousThreadId);
    }
    await loadLLMThreads(chat.id);
    await loadChatLLMConfigs();
    await loadMessages(chat.id);
    await loadChats(chat.id);
    sendPresence("view_thread", chat.id);
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
function sendPresence(action, threadId) {
    if (!wsConnected || !ws || ws.readyState !== WebSocket.OPEN) {
        return;
    }
    const parsedThreadId = Number(threadId || 0);
    if (action === "view_thread" && parsedThreadId <= 0) {
        return;
    }
    ws.send(JSON.stringify({
        type: "presence",
        action,
        thread_id: parsedThreadId > 0 ? parsedThreadId : undefined,
    }));
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
        if (activeThreadId) {
            sendPresence("view_thread", activeThreadId);
        }
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
                const handled = payload.user_id === "retry"
                    ? removeMessageIfNeeded(payload.message_id)
                    : markMessageRevoked(payload.message_id);
                if (handled) {
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
    const { response, data } = await sendMessage(activeThreadId, content, activeLLMThreadId);
    if (!response.ok) {
        chatSubtitle.textContent = data.error || "发送失败";
        if (data.code === "chat blocked") {
            activeChatBlocked = true;
            activeChatBlockMessage = data.error || "";
            messageInput.disabled = true;
        }
        else if (data.code === "chat reply required") {
            activeChatReplyRequired = Boolean(data.reply_required);
            activeChatReplyRequiredMessage = data.reply_required_message || data.error || "";
            messageInput.disabled = true;
        }
        return;
    }
    messageInput.value = "";
    activeChatReplyRequired = Boolean(data.reply_required);
    activeChatReplyRequiredMessage = data.reply_required_message || "";
    messageInput.disabled = activeChatBlocked || activeChatReplyRequired;
    updateActiveChatHeader();
    if (!wsConnected) {
        await loadMessages(activeThreadId);
    }
    await loadChats(activeThreadId);
});
attachmentInput.addEventListener("change", async () => {
    const file = attachmentInput.files?.[0];
    if (!file || !activeThreadId) {
        attachmentInput.value = "";
        return;
    }
    attachmentBtn.classList.add("uploading");
    chatSubtitle.textContent = t("chat.uploading");
    try {
        const { response, data } = await sendAttachment(activeThreadId, file);
        if (!response.ok) {
            chatSubtitle.textContent = data.error || t("chat.attachmentFailed");
            if (data.code === "chat blocked") {
                activeChatBlocked = true;
                activeChatBlockMessage = data.error || "";
                messageInput.disabled = true;
            }
            else if (data.code === "chat reply required") {
                activeChatReplyRequired = Boolean(data.reply_required);
                activeChatReplyRequiredMessage = data.reply_required_message || data.error || "";
                messageInput.disabled = true;
            }
            return;
        }
        activeChatReplyRequired = Boolean(data.reply_required);
        activeChatReplyRequiredMessage = data.reply_required_message || "";
        messageInput.disabled = activeChatBlocked || activeChatReplyRequired;
        updateActiveChatHeader();
        if (!wsConnected) {
            await loadMessages(activeThreadId);
        }
        await loadChats(activeThreadId);
    }
    finally {
        attachmentInput.value = "";
        attachmentBtn.classList.remove("uploading");
    }
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
        chatSubtitle.textContent = result.data.error || t("chat.createTopicFailed");
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
    const nextTitle = window.prompt(t("chat.renameTopicPrompt"), currentThread?.title || t("chat.newTopic"));
    if (nextTitle == null) {
        return;
    }
    const trimmed = nextTitle.trim();
    if (!trimmed) {
        chatSubtitle.textContent = t("chat.topicTitleEmpty");
        return;
    }
    const result = await updateLLMThread(activeThreadId, activeLLMThreadId, trimmed);
    if (!result.response.ok) {
        chatSubtitle.textContent = result.data.error || t("chat.renameFailed");
        return;
    }
    activeLLMThreads = result.data.threads || activeLLMThreads;
    renderLLMThreadBar();
    chatSubtitle.textContent = result.data.message || t("chat.renameSuccess");
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
        chatSubtitle.textContent = t("chat.selectModelFirst");
        return;
    }
    const result = await switchLLMThreadConfig(activeThreadId, activeLLMThreadId, llmConfigId);
    if (!result.response.ok) {
        chatSubtitle.textContent = result.data.error || t("chat.switchModelFailed");
        return;
    }
    activeLLMThreads = result.data.threads || activeLLMThreads;
    renderLLMThreadBar();
    chatSubtitle.textContent = result.data.message || t("chat.switchModelSuccess");
});
async function init() {
    await hydrateSiteBrand();
    await loadProfile();
    messageInput.disabled = true;
    connectWebSocket();
    const target = getTargetFromQuery();
    if (target.userId) {
        if (target.userId === currentUserId) {
            chatSubtitle.textContent = t("chat.cannotChatWithSelf");
            await loadChats();
            startPolling();
            return;
        }
        activeChatSummary = {
            id: "",
            other_user_id: target.userId,
            other_username: target.username || target.userId,
            other_user_online: false,
            other_user_device_type: "",
        };
        chatTitle.textContent = activeChatSummary.other_username;
        chatSubtitle.textContent = t("chat.loading");
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
window.addEventListener("beforeunload", () => {
    if (activeThreadId) {
        sendPresence("leave_thread", activeThreadId);
    }
});
