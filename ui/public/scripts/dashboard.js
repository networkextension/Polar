import { createBotUser, createLLMConfig, beginPasskeyRegistration, createTag, deleteApplePushCertificate, deleteEntry, fetchAvailableLLMConfigs, fetchBotUsers, fetchEntries, fetchEntry, fetchLLMConfigs, fetchLoginHistory, fetchPasskeys, fetchSiteSettings, fetchTags, finishPasskeyRegistration, removePasskey, removeBotUser, removeLLMConfig, removeTag, testLLMConfig, updateBotUser, updateLLMConfig, updateSiteSettings, updateTag, uploadApplePushCertificate, uploadSiteIcon, uploadUserIcon, } from "./api/dashboard.js";
import { fetchCurrentUser, logout } from "./api/session.js";
import { formatDeviceType } from "./lib/client.js";
import { makeDefaultAvatar } from "./lib/avatar.js";
import { byId, query } from "./lib/dom.js";
import { renderMarkdown } from "./lib/marked.js";
import { base64URLToBuffer, credentialToJSON } from "./lib/passkey.js";
import { hydrateSiteBrand, renderSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme, applyTheme } from "./lib/theme.js";
const welcomeText = byId("welcomeText");
const entryList = byId("entryList");
const entryContent = byId("entryContent");
const logoutBtn = byId("logoutBtn");
const newEntryBtn = byId("newEntryBtn");
const loadMoreBtn = byId("loadMoreBtn");
const editBtn = byId("editBtn");
const deleteBtn = byId("deleteBtn");
const drawerToggleBtn = byId("drawerToggleBtn");
const drawerCloseBtn = byId("drawerCloseBtn");
const drawerBackdrop = byId("drawerBackdrop");
const entryDrawer = byId("entryDrawer");
const loginHistoryList = byId("loginHistoryList");
const themeToggleBtn = byId("themeToggleBtn");
const passkeyRegisterBtn = byId("passkeyRegisterBtn");
const passkeyStatus = byId("passkeyStatus");
const passkeyList = byId("passkeyList");
const userIcon = byId("userIcon");
const iconFile = byId("iconFile");
const iconEditor = byId("iconEditor");
const iconCanvas = byId("iconCanvas");
const iconZoom = byId("iconZoom");
const saveIconBtn = byId("saveIconBtn");
const cancelIconBtn = byId("cancelIconBtn");
const iconStatus = byId("iconStatus");
const settingsCardAvatar = byId("settingsCardAvatar");
const settingsCardName = byId("settingsCardName");
const settingsCardMeta = byId("settingsCardMeta");
const siteManageCard = byId("siteManageCard");
const siteManageCardIcon = byId("siteManageCardIcon");
const siteManageCardName = byId("siteManageCardName");
const siteManageCardMeta = byId("siteManageCardMeta");
const settingsProfileName = byId("settingsProfileName");
const settingsProfileMeta = byId("settingsProfileMeta");
const addTagBtn = byId("addTagBtn");
const tagModal = byId("tagModal");
const tagModalTitle = byId("tagModalTitle");
const tagModalCloseBtn = byId("tagModalCloseBtn");
const tagForm = byId("tagForm");
const tagName = byId("tagName");
const tagSlug = byId("tagSlug");
const tagDesc = byId("tagDesc");
const tagOrder = byId("tagOrder");
const tagFormStatus = byId("tagFormStatus");
const tagSubmitBtn = byId("tagSubmitBtn");
const openSiteAdminBtn = byId("openSiteAdminBtn");
const siteAdminModal = byId("siteAdminModal");
const siteAdminModalTitle = byId("siteAdminModalTitle");
const siteAdminModalCloseBtn = byId("siteAdminModalCloseBtn");
const settingsSectionLead = byId("settingsSectionLead");
const siteAdminPanel = byId("siteAdminPanel");
const themeCurrentValue = byId("themeCurrentValue");
const llmConfigForm = byId("llmConfigForm");
const llmConfigNameInput = byId("llmConfigNameInput");
const llmConfigBaseUrlInput = byId("llmConfigBaseUrlInput");
const llmConfigModelInput = byId("llmConfigModelInput");
const llmConfigApiKeyInput = byId("llmConfigApiKeyInput");
const llmConfigSystemPromptInput = byId("llmConfigSystemPromptInput");
const llmConfigSharedInput = byId("llmConfigSharedInput");
const llmConfigResetBtn = byId("llmConfigResetBtn");
const llmConfigTestBtn = byId("llmConfigTestBtn");
const llmConfigSubmitBtn = byId("llmConfigSubmitBtn");
const llmConfigStatus = byId("llmConfigStatus");
const llmConfigList = byId("llmConfigList");
const settingsGitTagVersion = byId("settingsGitTagVersion");
const settingsOS = byId("settingsOS");
const settingsCPUArch = byId("settingsCPUArch");
const settingsPartitionCapacity = byId("settingsPartitionCapacity");
const settingsPartitionPath = byId("settingsPartitionPath");
const botUserForm = byId("botUserForm");
const botUserNameInput = byId("botUserNameInput");
const botUserConfigSelect = byId("botUserConfigSelect");
const botUserDescriptionInput = byId("botUserDescriptionInput");
const botUserSystemPromptInput = byId("botUserSystemPromptInput");
const botUserResetBtn = byId("botUserResetBtn");
const botUserSubmitBtn = byId("botUserSubmitBtn");
const botUserStatus = byId("botUserStatus");
const botUserList = byId("botUserList");
const siteNameInput = byId("siteNameInput");
const siteDescriptionInput = byId("siteDescriptionInput");
const saveSiteBtn = byId("saveSiteBtn");
const siteStatus = byId("siteStatus");
const siteIconPreview = byId("siteIconPreview");
const siteIconFile = byId("siteIconFile");
const applePushDevFile = byId("applePushDevFile");
const applePushProdFile = byId("applePushProdFile");
const applePushDevMeta = byId("applePushDevMeta");
const applePushProdMeta = byId("applePushProdMeta");
const applePushDevDeleteBtn = byId("applePushDevDeleteBtn");
const applePushProdDeleteBtn = byId("applePushProdDeleteBtn");
const siteAddTagBtnProxy = byId("siteAddTagBtnProxy");
const tagList = byId("tagList");
const settingsNavButtons = Array.from(document.querySelectorAll("[data-settings-nav]"));
const settingsPanels = Array.from(document.querySelectorAll("[data-settings-panel]"));
const settingsOpenButtons = Array.from(document.querySelectorAll("[data-open-settings-center], #openSiteAdminBtn"));
const iconCtx = iconCanvas.getContext("2d");
let nextOffset = 0;
let hasMore = true;
let activeEntryId = null;
let iconImage = null;
let baseScale = 1;
let zoomValue = 1;
let offsetX = 0;
let offsetY = 0;
let dragging = false;
let dragStartX = 0;
let dragStartY = 0;
let isAdmin = false;
let editingTagId = null;
let currentTags = [];
let editingLLMConfigId = null;
let editingBotUserId = null;
let currentLLMConfigs = [];
let currentAvailableLLMConfigs = [];
let currentBotUsers = [];
let activeSettingsSection = "personalization";
function setStatusMessage(element, message, tone = "default") {
    element.textContent = message;
    element.classList.remove("status-success", "status-error");
    if (tone === "success") {
        element.classList.add("status-success");
    }
    else if (tone === "error") {
        element.classList.add("status-error");
    }
}
function setModalOpen(modal, open) {
    modal.classList.toggle("open", open);
    modal.setAttribute("aria-hidden", open ? "false" : "true");
    if (modal === siteAdminModal || modal === tagModal) {
        document.body.classList.toggle("modal-open", open);
    }
}
function isMobileLayout() {
    return window.innerWidth <= 860;
}
function setDrawerOpen(open) {
    if (!isMobileLayout()) {
        entryDrawer.classList.remove("open");
        drawerBackdrop.classList.remove("open");
        return;
    }
    entryDrawer.classList.toggle("open", open);
    drawerBackdrop.classList.toggle("open", open);
}
function setActiveEntryItem() {
    entryList.querySelectorAll("li[data-entry-id]").forEach((item) => {
        item.classList.toggle("active", Number(item.dataset.entryId) === activeEntryId);
    });
}
function syncThemeButton(theme) {
    themeToggleBtn.textContent = theme === "mono" ? "切换到默认样式" : "切换到黑白样式";
    themeCurrentValue.textContent = theme === "mono" ? "黑白" : "默认";
}
function switchSettingsSection(section) {
    activeSettingsSection = section;
    const titles = {
        profile: {
            title: "个人中心",
            lead: "维护头像、登录方式和最近登录记录，把 Profile 与账户相关信息集中收好。",
        },
        personalization: {
            title: "个性化",
            lead: "管理界面风格和个人偏好，让工作台更贴近你的使用习惯。",
        },
        settings: {
            title: "设置",
            lead: "",
        },
        system: {
            title: "系统信息",
            lead: "查看当前实例的版本、系统环境与程序所在分区的剩余容量。",
        },
        bots: {
            title: "Bot 管理",
            lead: "在这里维护 Bot 的默认模型配置、简介和专属 Prompt。",
        },
        site: {
            title: "站点管理",
            lead: "管理员可见的站点 logo、intro、iOS Push 证书和 Tag 管理项。",
        },
    };
    settingsPanels.forEach((panel) => {
        const matched = panel.dataset.settingsPanel === section;
        panel.hidden = !matched;
        panel.classList.toggle("active", matched);
    });
    settingsNavButtons.forEach((button) => {
        button.classList.toggle("active", button.dataset.settingsNav === section);
    });
    siteAdminModalTitle.textContent = titles[section].title;
    settingsSectionLead.textContent = titles[section].lead;
}
function formatLocation(record) {
    const parts = [record.city, record.region, record.country].filter(Boolean);
    return parts.length > 0 ? parts.join(", ") : "位置未知";
}
function formatLoginMethod(method) {
    if (method === "passkey") {
        return "Passkey";
    }
    if (method === "register") {
        return "注册";
    }
    return "密码";
}
function formatPasskeyLabel(credentialId) {
    if (!credentialId) {
        return "Passkey";
    }
    if (credentialId.length <= 12) {
        return credentialId;
    }
    return `${credentialId.slice(0, 6)}...${credentialId.slice(-6)}`;
}
function renderPasskeys(credentials = []) {
    const count = credentials.length;
    if (!count) {
        setStatusMessage(passkeyStatus, "未绑定 Passkey。");
        passkeyRegisterBtn.textContent = "绑定 Passkey";
        passkeyList.innerHTML = "<li>还没有绑定任何 Passkey</li>";
        return;
    }
    setStatusMessage(passkeyStatus, `已绑定 ${count} 个 Passkey。`, "success");
    passkeyRegisterBtn.textContent = "继续绑定";
    passkeyList.innerHTML = credentials
        .map((item) => {
        const createdAt = new Date(item.created_at).toLocaleString();
        const updatedAt = new Date(item.updated_at).toLocaleString();
        return `
        <li>
          <div class="meta-title">已绑定 · ${formatPasskeyLabel(item.credential_id)}</div>
          <div class="meta-subtitle">创建时间：${createdAt}</div>
          <div class="meta-subtitle">最近更新时间：${updatedAt}</div>
          <div class="meta-time">
            <button class="btn-inline btn-secondary" type="button" data-passkey-delete="${item.credential_id}">删除</button>
          </div>
        </li>
      `;
    })
        .join("");
    passkeyList.querySelectorAll("[data-passkey-delete]").forEach((button) => {
        button.addEventListener("click", async () => {
            const credentialId = button.dataset.passkeyDelete || "";
            if (!credentialId) {
                return;
            }
            if (!window.confirm("确认删除这个 Passkey 吗？删除后将不能再用它快速登录。")) {
                return;
            }
            button.disabled = true;
            setStatusMessage(passkeyStatus, "正在删除 Passkey...");
            try {
                const { response, data } = await removePasskey(credentialId);
                if (!response.ok) {
                    setStatusMessage(passkeyStatus, data.error || "删除失败", "error");
                    button.disabled = false;
                    return;
                }
                renderPasskeys(data.credentials || []);
                setStatusMessage(passkeyStatus, data.message || "Passkey 已删除", "success");
            }
            catch {
                setStatusMessage(passkeyStatus, "网络错误，请重试", "error");
                button.disabled = false;
            }
        });
    });
}
function defaultSiteIcon(name) {
    return makeDefaultAvatar(name || "站", 160);
}
function formatCertificateMeta(cert) {
    if (!cert?.file_name) {
        return "未上传";
    }
    const uploadedAt = cert.uploaded_at ? new Date(cert.uploaded_at).toLocaleString() : "未知时间";
    return `当前文件：${cert.file_name} · 上传时间：${uploadedAt}`;
}
function renderSystemInfo(systemInfo) {
    settingsGitTagVersion.textContent = systemInfo?.git_tag_version || "未知";
    settingsOS.textContent = systemInfo?.os || "未知";
    settingsCPUArch.textContent = systemInfo?.cpu_arch || "未知";
    settingsPartitionCapacity.textContent = systemInfo?.partition_capacity || "未知";
    settingsPartitionPath.textContent = systemInfo?.partition_path
        ? `路径：${systemInfo.partition_path}`
        : "";
}
function renderSiteSettings(site) {
    const safeSite = site || { name: "Polar-", description: "", icon_url: "" };
    siteNameInput.value = safeSite.name || "Polar-";
    siteDescriptionInput.value = safeSite.description || "";
    siteIconPreview.src = safeSite.icon_url || defaultSiteIcon(safeSite.name || "Polar-");
    siteManageCardIcon.src = safeSite.icon_url || defaultSiteIcon(safeSite.name || "站");
    siteManageCardName.textContent = safeSite.name || "站点管理";
    siteManageCardMeta.textContent = safeSite.description || "维护站点信息、证书与 Tag";
    applePushDevMeta.textContent = formatCertificateMeta(safeSite.apple_push_dev_cert);
    applePushProdMeta.textContent = formatCertificateMeta(safeSite.apple_push_prod_cert);
    applePushDevDeleteBtn.disabled = !safeSite.apple_push_dev_cert?.file_url;
    applePushProdDeleteBtn.disabled = !safeSite.apple_push_prod_cert?.file_url;
    renderSystemInfo(safeSite.system_info);
    renderSiteBrand(safeSite);
}
function renderTagList(tags) {
    if (!tags.length) {
        tagList.innerHTML = `<li class="tag-item tag-item-empty">还没有 Tag，先创建第一个吧。</li>`;
        return;
    }
    tagList.innerHTML = tags
        .map((tag) => `
        <li class="tag-item" data-tag-id="${tag.id}">
          <div class="tag-item-main">
            <div class="tag-item-header">
              <strong>${tag.name}</strong>
              <span class="tag-chip">${tag.slug}</span>
            </div>
            <div class="tag-item-meta">排序 ${tag.sort_order}</div>
            <div class="tag-item-desc">${tag.description || "暂无描述"}</div>
          </div>
          <div class="tag-item-actions">
            <button class="btn-inline btn-secondary" type="button" data-action="edit">编辑</button>
            <button class="btn-inline" type="button" data-action="delete">删除</button>
          </div>
        </li>
      `)
        .join("");
}
function resetLLMConfigForm() {
    editingLLMConfigId = null;
    llmConfigForm.reset();
    llmConfigApiKeyInput.value = "";
    llmConfigStatus.textContent = "";
    llmConfigSubmitBtn.textContent = "保存配置";
}
function resetBotUserForm() {
    editingBotUserId = null;
    botUserForm.reset();
    botUserStatus.textContent = "";
    botUserSubmitBtn.textContent = "保存 Bot";
}
function renderLLMConfigList(configs) {
    if (!configs.length) {
        llmConfigList.innerHTML = `<li class="tag-item tag-item-empty">还没有 LLM Config，先创建一个吧。</li>`;
        return;
    }
    llmConfigList.innerHTML = configs
        .map((config) => `
        <li class="tag-item" data-llm-config-id="${config.id}">
          <div class="tag-item-main">
            <div class="tag-item-header">
              <strong>${config.name}</strong>
              <span class="tag-chip">${config.model}</span>
              ${config.has_api_key ? `<span class="tag-chip">Key 已保存</span>` : `<span class="tag-chip">无 Key</span>`}
              ${config.shared ? `<span class="tag-chip">共享</span>` : `<span class="tag-chip">私有</span>`}
            </div>
            <div class="tag-item-meta">${config.base_url}</div>
            <div class="tag-item-desc">${config.system_prompt || "未设置 System Prompt"}</div>
          </div>
          <div class="tag-item-actions">
            <button class="btn-inline btn-secondary" type="button" data-action="edit">编辑</button>
            <button class="btn-inline" type="button" data-action="delete">删除</button>
          </div>
        </li>
      `)
        .join("");
}
function syncBotConfigOptions(configs) {
    if (!configs.length) {
        botUserConfigSelect.innerHTML = `<option value="">请先创建 LLM Config</option>`;
        botUserConfigSelect.disabled = true;
        botUserSubmitBtn.disabled = true;
        return;
    }
    const currentValue = editingBotUserId ? botUserConfigSelect.value : "";
    botUserConfigSelect.innerHTML = configs
        .map((config) => `<option value="${config.id}">${config.name} · ${config.model}</option>`)
        .join("");
    if (currentValue && configs.some((config) => String(config.id) === currentValue)) {
        botUserConfigSelect.value = currentValue;
    }
    botUserConfigSelect.disabled = false;
    botUserSubmitBtn.disabled = false;
}
function renderBotUserList(bots) {
    if (!bots.length) {
        botUserList.innerHTML = `<li class="tag-item tag-item-empty">还没有 Bot，创建后就能直接私聊使用。</li>`;
        return;
    }
    botUserList.innerHTML = bots
        .map((bot) => `
        <li class="tag-item" data-bot-user-id="${bot.id}">
          <div class="tag-item-main">
            <div class="tag-item-header">
              <strong>${bot.name}</strong>
              <span class="tag-chip">${bot.config_name}</span>
            </div>
            <div class="tag-item-meta">用户 ID：${bot.bot_user_id}</div>
            <div class="tag-item-desc">${bot.description || "暂无简介"}</div>
            <div class="tag-item-meta">${bot.system_prompt ? `Bot Prompt：${bot.system_prompt.slice(0, 36)}${bot.system_prompt.length > 36 ? "..." : ""}` : "未配置 Bot Prompt"}</div>
          </div>
          <div class="tag-item-actions">
            <button class="btn-inline btn-secondary" type="button" data-action="chat">对话</button>
            <button class="btn-inline btn-secondary" type="button" data-action="edit">编辑</button>
            <button class="btn-inline" type="button" data-action="delete">删除</button>
          </div>
        </li>
      `)
        .join("");
}
async function loadLoginHistory() {
    const { response, data } = await fetchLoginHistory();
    if (!response.ok) {
        loginHistoryList.innerHTML = "<li>无法加载登录记录</li>";
        return;
    }
    const records = data.records || [];
    if (!records.length) {
        loginHistoryList.innerHTML = "<li>暂无登录记录</li>";
        return;
    }
    loginHistoryList.innerHTML = records
        .map((record) => {
        const time = new Date(record.logged_in_at).toLocaleString();
        return `
        <li>
          <div class="meta-title">${record.ip_address || "未知 IP"} · ${formatLoginMethod(record.login_method)} · ${formatDeviceType(record.device_type)}</div>
          <div class="meta-subtitle">${formatLocation(record)}</div>
          <div class="meta-time">${time}</div>
        </li>
      `;
    })
        .join("");
}
async function loadPasskeys() {
    const { response, data } = await fetchPasskeys();
    if (!response.ok) {
        setStatusMessage(passkeyStatus, data.error || "无法加载 Passkey 状态", "error");
        passkeyList.innerHTML = "<li>无法加载 Passkey 列表</li>";
        return;
    }
    renderPasskeys(data.credentials || []);
}
async function loadProfile() {
    const { response, data } = await fetchCurrentUser();
    if (!response.ok) {
        window.location.href = "/login.html";
        return;
    }
    isAdmin = data.role === "admin";
    welcomeText.textContent = `你好，${data.username}`;
    settingsCardName.textContent = data.username || "当前用户";
    settingsCardMeta.textContent = isAdmin ? "Profile、个性化与站点管理" : "Profile 与个性化";
    settingsProfileName.textContent = data.username || "当前用户";
    settingsProfileMeta.textContent = isAdmin ? "维护你的个人资料、设备偏好与站点配置" : "维护你的个人资料、设备与偏好";
    addTagBtn.disabled = !isAdmin;
    addTagBtn.textContent = isAdmin ? "新建 Tag" : "仅管理员可新建 Tag";
    addTagBtn.hidden = !isAdmin;
    siteManageCard.hidden = !isAdmin;
    siteAdminPanel.hidden = !isAdmin;
    settingsNavButtons.forEach((button) => {
        if (button.dataset.settingsNav === "site") {
            button.hidden = !isAdmin;
        }
    });
    const avatar = data.icon_url || makeDefaultAvatar(data.username || "U", 160);
    if (data.icon_url) {
        userIcon.src = data.icon_url;
    }
    userIcon.src = avatar;
    settingsCardAvatar.src = avatar;
}
async function loadSiteAdminData() {
    const tasks = [
        (async () => {
            const { response, data } = await fetchSiteSettings();
            if (response.ok) {
                renderSiteSettings(data.site);
                return;
            }
            renderSystemInfo();
            if (isAdmin) {
                siteStatus.textContent = data.error || "无法加载站点信息";
            }
        })(),
        (async () => {
            const [ownResult, availableResult] = await Promise.all([fetchLLMConfigs(), fetchAvailableLLMConfigs()]);
            if (ownResult.response.ok) {
                currentLLMConfigs = ownResult.data.configs || [];
                renderLLMConfigList(currentLLMConfigs);
            }
            else {
                llmConfigStatus.textContent = ownResult.data.error || "无法加载 LLM Config";
                llmConfigList.innerHTML = `<li class="tag-item tag-item-empty">无法加载 LLM Config</li>`;
            }
            if (availableResult.response.ok) {
                currentAvailableLLMConfigs = availableResult.data.configs || [];
                syncBotConfigOptions(currentAvailableLLMConfigs);
            }
            else {
                currentAvailableLLMConfigs = [];
                botUserStatus.textContent = availableResult.data.error || "无法加载可用配置";
                syncBotConfigOptions(currentAvailableLLMConfigs);
            }
        })(),
        (async () => {
            const { response, data } = await fetchBotUsers();
            if (response.ok) {
                currentBotUsers = data.bots || [];
                renderBotUserList(currentBotUsers);
            }
            else {
                botUserStatus.textContent = data.error || "无法加载 Bot 列表";
                botUserList.innerHTML = `<li class="tag-item tag-item-empty">无法加载 Bot 列表</li>`;
            }
        })(),
    ];
    if (isAdmin) {
        tasks.push((async () => {
            const { response, data } = await fetchTags();
            if (response.ok) {
                currentTags = data.tags || [];
                renderTagList(currentTags);
            }
            else {
                tagList.innerHTML = `<li class="tag-item tag-item-empty">无法加载 Tag 列表</li>`;
            }
        })());
    }
    await Promise.all(tasks);
}
function openTagModal(tag) {
    editingTagId = tag?.id || null;
    tagForm.reset();
    tagOrder.value = String(tag?.sort_order ?? 0);
    tagName.value = tag?.name || "";
    tagSlug.value = tag?.slug || "";
    tagDesc.value = tag?.description || "";
    tagFormStatus.textContent = "";
    tagModalTitle.textContent = editingTagId ? "编辑 Tag" : "添加 Tag";
    tagSubmitBtn.textContent = editingTagId ? "保存" : "创建";
    setModalOpen(tagModal, true);
    tagName.focus();
}
function closeTagModal() {
    editingTagId = null;
    setModalOpen(tagModal, false);
}
function openSiteAdminModal(section = "personalization") {
    switchSettingsSection(section);
    setModalOpen(siteAdminModal, true);
    if (section === "settings") {
        llmConfigNameInput.focus();
        return;
    }
    if (section === "system") {
        settingsSectionLead.focus?.();
        return;
    }
    if (section === "profile") {
        passkeyRegisterBtn.focus();
        return;
    }
    themeToggleBtn.focus();
}
function closeSiteAdminModal() {
    setModalOpen(siteAdminModal, false);
}
async function loadEntries(reset = false) {
    if (reset) {
        nextOffset = 0;
        hasMore = true;
        entryList.innerHTML = "";
    }
    if (!hasMore) {
        return;
    }
    const { response, data } = await fetchEntries(nextOffset);
    if (!response.ok) {
        entryList.innerHTML = "<li>无法加载记录</li>";
        return;
    }
    const entries = data.entries || [];
    if (reset && !entries.length) {
        entryList.innerHTML = "<li>暂无记录</li>";
        hasMore = false;
        loadMoreBtn.style.display = "none";
        return;
    }
    entries.forEach((entry) => {
        const li = document.createElement("li");
        li.dataset.entryId = String(entry.id);
        li.innerHTML = entry.is_public
            ? `<span>${entry.title}</span><span class="tag-chip">Public</span>`
            : `<span>${entry.title}</span>`;
        li.addEventListener("click", () => {
            void loadEntry(entry.id);
        });
        entryList.appendChild(li);
    });
    hasMore = Boolean(data.has_more);
    nextOffset = Number(data.next_offset || 0);
    loadMoreBtn.style.display = hasMore ? "inline-flex" : "none";
}
async function loadEntry(id) {
    const { response, data } = await fetchEntry(id);
    if (!response.ok) {
        entryContent.textContent = "读取失败";
        return;
    }
    activeEntryId = data.entry ? data.entry.id : null;
    setActiveEntryItem();
    const rawContent = data.content || "空内容";
    entryContent.innerHTML = renderMarkdown(rawContent);
    editBtn.disabled = data.can_edit === false;
    deleteBtn.disabled = data.can_edit === false;
    if (isMobileLayout()) {
        setDrawerOpen(false);
    }
}
function drawIconPreview() {
    if (!iconCtx || !iconImage) {
        return;
    }
    const scale = baseScale * zoomValue;
    const drawW = iconImage.width * scale;
    const drawH = iconImage.height * scale;
    const minX = iconCanvas.width - drawW;
    const minY = iconCanvas.height - drawH;
    offsetX = Math.min(0, Math.max(minX, offsetX));
    offsetY = Math.min(0, Math.max(minY, offsetY));
    iconCtx.clearRect(0, 0, iconCanvas.width, iconCanvas.height);
    iconCtx.drawImage(iconImage, (iconCanvas.width - drawW) / 2 + offsetX, (iconCanvas.height - drawH) / 2 + offsetY, drawW, drawH);
}
function startDrag(clientX, clientY) {
    dragging = true;
    dragStartX = clientX;
    dragStartY = clientY;
}
function moveDrag(clientX, clientY) {
    if (!dragging) {
        return;
    }
    offsetX += clientX - dragStartX;
    offsetY += clientY - dragStartY;
    dragStartX = clientX;
    dragStartY = clientY;
    drawIconPreview();
}
function stopDrag() {
    dragging = false;
}
addTagBtn.addEventListener("click", () => {
    if (addTagBtn.disabled) {
        return;
    }
    openTagModal();
});
settingsOpenButtons.forEach((button) => {
    button.addEventListener("click", () => {
        const target = button.dataset.settingsTarget || "personalization";
        openSiteAdminModal(target);
    });
});
siteAdminModalCloseBtn.addEventListener("click", closeSiteAdminModal);
query(siteAdminModal, ".modal-backdrop").addEventListener("click", closeSiteAdminModal);
settingsNavButtons.forEach((button) => {
    button.addEventListener("click", () => {
        const target = button.dataset.settingsNav;
        if (!target) {
            return;
        }
        switchSettingsSection(target);
    });
});
siteAddTagBtnProxy.addEventListener("click", () => {
    if (!isAdmin) {
        return;
    }
    openTagModal();
});
tagModalCloseBtn.addEventListener("click", closeTagModal);
query(tagModal, ".modal-backdrop").addEventListener("click", closeTagModal);
tagForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    tagFormStatus.textContent = editingTagId ? "正在保存..." : "正在创建...";
    tagSubmitBtn.disabled = true;
    const payload = {
        name: tagName.value.trim(),
        slug: tagSlug.value.trim(),
        description: tagDesc.value.trim(),
        sort_order: Number(tagOrder.value || 0),
    };
    try {
        const { response, data } = editingTagId
            ? await updateTag(editingTagId, payload)
            : await createTag(payload);
        if (!response.ok) {
            tagFormStatus.textContent = data.error || "保存失败";
            return;
        }
        tagFormStatus.textContent = editingTagId ? "保存成功" : "创建成功";
        await loadSiteAdminData();
        window.setTimeout(() => {
            closeTagModal();
        }, 400);
    }
    catch {
        tagFormStatus.textContent = "保存失败，请重试";
    }
    finally {
        tagSubmitBtn.disabled = false;
    }
});
tagList.addEventListener("click", async (event) => {
    const target = event.target;
    const button = target.closest("button[data-action]");
    if (!button) {
        return;
    }
    const item = button.closest("[data-tag-id]");
    const tagId = Number(item?.dataset.tagId || 0);
    const tag = currentTags.find((entry) => entry.id === tagId);
    if (!tag) {
        return;
    }
    const action = button.dataset.action;
    if (action === "edit") {
        openTagModal(tag);
        return;
    }
    if (action === "delete") {
        if (!window.confirm(`确定删除 Tag “${tag.name}” 吗？`)) {
            return;
        }
        const { response, data } = await removeTag(tag.id);
        if (!response.ok) {
            siteStatus.textContent = data.error || "删除失败";
            return;
        }
        siteStatus.textContent = `已删除 Tag：${tag.name}`;
        await loadSiteAdminData();
    }
});
llmConfigResetBtn.addEventListener("click", () => {
    resetLLMConfigForm();
});
llmConfigTestBtn.addEventListener("click", async () => {
    const payload = {
        name: llmConfigNameInput.value.trim(),
        base_url: llmConfigBaseUrlInput.value.trim(),
        model: llmConfigModelInput.value.trim(),
        api_key: llmConfigApiKeyInput.value.trim(),
        system_prompt: llmConfigSystemPromptInput.value.trim(),
        shared: llmConfigSharedInput.checked,
    };
    if (!payload.base_url || !payload.model || !payload.api_key) {
        setStatusMessage(llmConfigStatus, "测试前请先填写 Base URL、Model 和 API Key", "error");
        return;
    }
    llmConfigTestBtn.disabled = true;
    llmConfigSubmitBtn.disabled = true;
    setStatusMessage(llmConfigStatus, "正在测试模型配置...");
    try {
        const { response, data } = await testLLMConfig(payload);
        setStatusMessage(llmConfigStatus, response.ok ? data.message || "连接成功，模型配置可用" : data.error || "测试失败", response.ok ? "success" : "error");
    }
    catch {
        setStatusMessage(llmConfigStatus, "网络错误，请重试", "error");
    }
    finally {
        llmConfigTestBtn.disabled = false;
        llmConfigSubmitBtn.disabled = false;
    }
});
botUserResetBtn.addEventListener("click", () => {
    resetBotUserForm();
});
llmConfigForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
        name: llmConfigNameInput.value.trim(),
        base_url: llmConfigBaseUrlInput.value.trim(),
        model: llmConfigModelInput.value.trim(),
        api_key: llmConfigApiKeyInput.value.trim(),
        system_prompt: llmConfigSystemPromptInput.value.trim(),
        shared: llmConfigSharedInput.checked,
    };
    if (!payload.name || !payload.base_url || !payload.model) {
        llmConfigStatus.textContent = "请先填写名称、Base URL 和 Model";
        return;
    }
    llmConfigSubmitBtn.disabled = true;
    llmConfigStatus.textContent = editingLLMConfigId ? "正在保存配置..." : "正在创建配置...";
    try {
        const { response, data } = editingLLMConfigId
            ? await updateLLMConfig(editingLLMConfigId, {
                ...payload,
                update_api_key: payload.api_key !== "",
            })
            : await createLLMConfig(payload);
        if (!response.ok) {
            llmConfigStatus.textContent = data.error || "保存失败";
            return;
        }
        llmConfigStatus.textContent = editingLLMConfigId ? "配置已更新" : "配置已创建";
        resetLLMConfigForm();
        await loadSiteAdminData();
    }
    catch {
        llmConfigStatus.textContent = "网络错误，请重试";
    }
    finally {
        llmConfigSubmitBtn.disabled = false;
    }
});
llmConfigList.addEventListener("click", async (event) => {
    const target = event.target;
    const button = target.closest("button[data-action]");
    if (!button) {
        return;
    }
    const item = button.closest("[data-llm-config-id]");
    const configID = Number(item?.dataset.llmConfigId || 0);
    const config = currentLLMConfigs.find((entry) => entry.id === configID);
    if (!config) {
        return;
    }
    const action = button.dataset.action;
    if (action === "edit") {
        editingLLMConfigId = config.id;
        llmConfigNameInput.value = config.name;
        llmConfigBaseUrlInput.value = config.base_url;
        llmConfigModelInput.value = config.model;
        llmConfigApiKeyInput.value = "";
        llmConfigSystemPromptInput.value = config.system_prompt || "";
        llmConfigSharedInput.checked = config.shared;
        llmConfigSubmitBtn.textContent = "更新配置";
        llmConfigStatus.textContent = config.has_api_key ? "已保存 API Key，留空表示保持原值" : "";
        llmConfigNameInput.focus();
        return;
    }
    if (action === "delete") {
        if (!window.confirm(`确定删除配置“${config.name}”吗？已绑定的 Bot 需要先改绑或删除。`)) {
            return;
        }
        const { response, data } = await removeLLMConfig(config.id);
        if (!response.ok) {
            llmConfigStatus.textContent = data.error || "删除失败";
            return;
        }
        llmConfigStatus.textContent = `已删除配置：${config.name}`;
        resetLLMConfigForm();
        await loadSiteAdminData();
    }
});
botUserForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
        name: botUserNameInput.value.trim(),
        description: botUserDescriptionInput.value.trim(),
        system_prompt: botUserSystemPromptInput.value.trim(),
        llm_config_id: Number(botUserConfigSelect.value || 0),
    };
    if (!payload.name || payload.llm_config_id <= 0) {
        botUserStatus.textContent = "请先填写 Bot 名称并选择一个配置";
        return;
    }
    botUserSubmitBtn.disabled = true;
    botUserStatus.textContent = editingBotUserId ? "正在保存 Bot..." : "正在创建 Bot...";
    try {
        const { response, data } = editingBotUserId
            ? await updateBotUser(editingBotUserId, payload)
            : await createBotUser(payload);
        if (!response.ok) {
            botUserStatus.textContent = data.error || "保存失败";
            return;
        }
        botUserStatus.textContent = editingBotUserId ? "Bot 已更新" : "Bot 已创建";
        resetBotUserForm();
        await loadSiteAdminData();
    }
    catch {
        botUserStatus.textContent = "网络错误，请重试";
    }
    finally {
        botUserSubmitBtn.disabled = false;
    }
});
botUserList.addEventListener("click", async (event) => {
    const target = event.target;
    const button = target.closest("button[data-action]");
    if (!button) {
        return;
    }
    const item = button.closest("[data-bot-user-id]");
    const botID = Number(item?.dataset.botUserId || 0);
    const bot = currentBotUsers.find((entry) => entry.id === botID);
    if (!bot) {
        return;
    }
    const action = button.dataset.action;
    if (action === "chat") {
        window.location.href = `/chat.html?user_id=${encodeURIComponent(bot.bot_user_id)}&username=${encodeURIComponent(bot.name)}`;
        return;
    }
    if (action === "edit") {
        editingBotUserId = bot.id;
        botUserNameInput.value = bot.name;
        botUserDescriptionInput.value = bot.description || "";
        botUserSystemPromptInput.value = bot.system_prompt || "";
        botUserConfigSelect.value = String(bot.llm_config_id);
        botUserSubmitBtn.textContent = "更新 Bot";
        botUserStatus.textContent = "";
        botUserNameInput.focus();
        return;
    }
    if (action === "delete") {
        if (!window.confirm(`确定删除 Bot “${bot.name}”吗？这会同时移除这个 Bot 用户。`)) {
            return;
        }
        const { response, data } = await removeBotUser(bot.id);
        if (!response.ok) {
            botUserStatus.textContent = data.error || "删除失败";
            return;
        }
        botUserStatus.textContent = `已删除 Bot：${bot.name}`;
        resetBotUserForm();
        await loadSiteAdminData();
    }
});
saveSiteBtn.addEventListener("click", async () => {
    siteStatus.textContent = "正在保存站点信息...";
    saveSiteBtn.disabled = true;
    try {
        const { response, data } = await updateSiteSettings({
            name: siteNameInput.value.trim(),
            description: siteDescriptionInput.value.trim(),
            icon_url: siteIconPreview.src,
        });
        if (!response.ok) {
            siteStatus.textContent = data.error || "保存失败";
            return;
        }
        renderSiteSettings(data.site);
        siteStatus.textContent = "站点信息已保存";
    }
    catch {
        siteStatus.textContent = "网络错误，请重试";
    }
    finally {
        saveSiteBtn.disabled = false;
    }
});
siteIconFile.addEventListener("change", async () => {
    const file = siteIconFile.files?.[0];
    if (!file) {
        return;
    }
    if (!file.type.startsWith("image/")) {
        siteStatus.textContent = "请选择图片文件";
        siteIconFile.value = "";
        return;
    }
    siteStatus.textContent = "正在上传站点图标...";
    const formData = new FormData();
    formData.append("icon", file);
    try {
        const { response, data } = await uploadSiteIcon(formData);
        if (!response.ok) {
            siteStatus.textContent = data.error || "上传失败";
            return;
        }
        renderSiteSettings(data.site);
        siteIconPreview.src = `${data.icon_url || data.site?.icon_url || ""}?v=${Date.now()}`;
        siteStatus.textContent = "站点图标已更新";
    }
    catch {
        siteStatus.textContent = "网络错误，请重试";
    }
    finally {
        siteIconFile.value = "";
    }
});
async function handleApplePushCertificateUpload(environment, fileInput) {
    const file = fileInput.files?.[0];
    if (!file) {
        return;
    }
    const lowerName = file.name.toLowerCase();
    if (![".p8", ".p12", ".pem", ".cer", ".crt", ".key"].some((ext) => lowerName.endsWith(ext))) {
        siteStatus.textContent = "仅支持 .p8、.p12、.pem、.cer、.crt、.key 文件";
        fileInput.value = "";
        return;
    }
    siteStatus.textContent = `正在上传 ${environment} Apple Push 证书...`;
    const formData = new FormData();
    formData.append("certificate", file);
    try {
        const { response, data } = await uploadApplePushCertificate(environment, formData);
        if (!response.ok) {
            siteStatus.textContent = data.error || "上传失败";
            return;
        }
        renderSiteSettings(data.site);
        siteStatus.textContent = `${environment} Apple Push 证书已更新`;
    }
    catch {
        siteStatus.textContent = "网络错误，请重试";
    }
    finally {
        fileInput.value = "";
    }
}
async function handleApplePushCertificateDelete(environment) {
    if (!window.confirm(`确定删除 ${environment} Apple Push 证书吗？`)) {
        return;
    }
    siteStatus.textContent = `正在删除 ${environment} Apple Push 证书...`;
    try {
        const { response, data } = await deleteApplePushCertificate(environment);
        if (!response.ok) {
            siteStatus.textContent = data.error || "删除失败";
            return;
        }
        renderSiteSettings(data.site);
        siteStatus.textContent = `${environment} Apple Push 证书已删除`;
    }
    catch {
        siteStatus.textContent = "网络错误，请重试";
    }
}
applePushDevFile.addEventListener("change", async () => {
    await handleApplePushCertificateUpload("dev", applePushDevFile);
});
applePushProdFile.addEventListener("change", async () => {
    await handleApplePushCertificateUpload("prod", applePushProdFile);
});
applePushDevDeleteBtn.addEventListener("click", async () => {
    await handleApplePushCertificateDelete("dev");
});
applePushProdDeleteBtn.addEventListener("click", async () => {
    await handleApplePushCertificateDelete("prod");
});
logoutBtn.addEventListener("click", async () => {
    logoutBtn.disabled = true;
    try {
        const response = await logout();
        if (!response.ok) {
            welcomeText.textContent = "退出失败，请重试";
            return;
        }
        window.location.replace("/login.html");
    }
    catch {
        welcomeText.textContent = "退出失败，请检查网络后重试";
    }
    finally {
        logoutBtn.disabled = false;
    }
});
newEntryBtn.addEventListener("click", () => {
    window.location.href = "/editor.html";
});
loadMoreBtn.addEventListener("click", () => {
    void loadEntries();
});
drawerToggleBtn.addEventListener("click", () => {
    setDrawerOpen(!entryDrawer.classList.contains("open"));
});
drawerCloseBtn.addEventListener("click", () => {
    setDrawerOpen(false);
});
drawerBackdrop.addEventListener("click", () => {
    setDrawerOpen(false);
});
window.addEventListener("resize", () => {
    if (!isMobileLayout()) {
        setDrawerOpen(false);
    }
});
editBtn.addEventListener("click", () => {
    if (!activeEntryId) {
        return;
    }
    window.location.href = `/editor.html?id=${activeEntryId}`;
});
deleteBtn.addEventListener("click", async () => {
    if (!activeEntryId) {
        return;
    }
    if (!window.confirm("确定要删除该记录吗？")) {
        return;
    }
    const response = await deleteEntry(activeEntryId);
    if (response.ok) {
        activeEntryId = null;
        entryContent.textContent = "请选择侧边栏中的记录";
        await loadEntries(true);
    }
});
iconFile.addEventListener("change", () => {
    const file = iconFile.files?.[0];
    if (!file) {
        return;
    }
    if (!file.type.startsWith("image/")) {
        iconStatus.textContent = "请选择图片文件。";
        return;
    }
    const reader = new FileReader();
    reader.onload = () => {
        const result = reader.result;
        if (typeof result !== "string") {
            return;
        }
        const img = new Image();
        img.onload = () => {
            iconImage = img;
            baseScale = Math.max(iconCanvas.width / img.width, iconCanvas.height / img.height);
            zoomValue = 1;
            offsetX = 0;
            offsetY = 0;
            iconZoom.value = "1";
            iconEditor.classList.add("active");
            drawIconPreview();
        };
        img.src = result;
    };
    reader.readAsDataURL(file);
});
iconZoom.addEventListener("input", () => {
    zoomValue = Number(iconZoom.value);
    drawIconPreview();
});
iconCanvas.addEventListener("mousedown", (event) => {
    startDrag(event.clientX, event.clientY);
});
window.addEventListener("mousemove", (event) => {
    moveDrag(event.clientX, event.clientY);
});
window.addEventListener("mouseup", stopDrag);
iconCanvas.addEventListener("touchstart", (event) => {
    const touch = event.touches[0];
    if (!touch) {
        return;
    }
    startDrag(touch.clientX, touch.clientY);
});
window.addEventListener("touchmove", (event) => {
    const touch = event.touches[0];
    if (!touch) {
        return;
    }
    moveDrag(touch.clientX, touch.clientY);
});
window.addEventListener("touchend", stopDrag);
window.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") {
        return;
    }
    if (tagModal.classList.contains("open")) {
        closeTagModal();
    }
    if (siteAdminModal.classList.contains("open")) {
        closeSiteAdminModal();
    }
});
cancelIconBtn.addEventListener("click", () => {
    iconEditor.classList.remove("active");
    iconFile.value = "";
    iconStatus.textContent = "已取消编辑。";
});
saveIconBtn.addEventListener("click", async () => {
    if (!iconImage) {
        return;
    }
    iconStatus.textContent = "正在上传...";
    iconCanvas.toBlob(async (blob) => {
        if (!blob) {
            iconStatus.textContent = "生成图片失败。";
            return;
        }
        const formData = new FormData();
        formData.append("icon", blob, "icon.png");
        try {
            const { response, data } = await uploadUserIcon(formData);
            if (!response.ok) {
                iconStatus.textContent = data.error || "上传失败";
                return;
            }
            const nextAvatar = `${data.icon_url || ""}?v=${Date.now()}`;
            userIcon.src = nextAvatar;
            settingsCardAvatar.src = nextAvatar;
            iconEditor.classList.remove("active");
            iconFile.value = "";
            iconStatus.textContent = "头像已更新。";
        }
        catch {
            iconStatus.textContent = "网络错误，请重试。";
        }
    }, "image/png", 0.92);
});
themeToggleBtn.addEventListener("click", () => {
    const currentTheme = document.documentElement.dataset.theme === "mono" ? "mono" : "default";
    const nextTheme = applyTheme(currentTheme === "mono" ? "default" : "mono", true);
    syncThemeButton(nextTheme);
});
passkeyRegisterBtn.addEventListener("click", async () => {
    if (!window.PublicKeyCredential) {
        setStatusMessage(passkeyStatus, "当前浏览器不支持 Passkey。", "error");
        return;
    }
    setStatusMessage(passkeyStatus, "正在启动 Passkey...");
    try {
        const { response: beginResponse, data: beginResult } = await beginPasskeyRegistration();
        if (!beginResponse.ok) {
            setStatusMessage(passkeyStatus, beginResult.error || "无法发起 Passkey 绑定", "error");
            return;
        }
        const publicKey = beginResult.publicKey;
        publicKey.challenge = base64URLToBuffer(publicKey.challenge);
        publicKey.user.id = base64URLToBuffer(publicKey.user.id);
        if (publicKey.excludeCredentials) {
            publicKey.excludeCredentials = publicKey.excludeCredentials.map((cred) => ({
                ...cred,
                id: base64URLToBuffer(cred.id),
            }));
        }
        const credential = await navigator.credentials.create({
            publicKey: publicKey,
        });
        const payload = credentialToJSON(credential);
        const { response: finishResponse, data: finishResult } = await finishPasskeyRegistration(beginResult.session_id || "", payload);
        if (!finishResponse.ok) {
            setStatusMessage(passkeyStatus, finishResult.error || "Passkey 绑定失败", "error");
            return;
        }
        renderPasskeys(finishResult.credentials || []);
        setStatusMessage(passkeyStatus, finishResult.message || "Passkey 绑定成功！", "success");
    }
    catch {
        setStatusMessage(passkeyStatus, "网络错误，请重试", "error");
    }
});
const initialTheme = initStoredTheme();
syncThemeButton(initialTheme);
bindThemeSync(syncThemeButton);
switchSettingsSection(activeSettingsSection);
void (async () => {
    await hydrateSiteBrand();
    await loadProfile();
    await Promise.all([loadEntries(true), loadLoginHistory(), loadPasskeys(), loadSiteAdminData()]);
})();
