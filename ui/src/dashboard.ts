import {
  createBotUser,
  createLLMConfig,
  beginPasskeyRegistration,
  createTag,
  deleteApplePushCertificate,
  deleteEntry,
  fetchBotUsers,
  fetchEntries,
  fetchEntry,
  fetchLLMConfigs,
  fetchLoginHistory,
  fetchSiteSettings,
  fetchTags,
  finishPasskeyRegistration,
  removeBotUser,
  removeLLMConfig,
  removeTag,
  testLLMConfig,
  updateBotUser,
  updateLLMConfig,
  updateSiteSettings,
  updateTag,
  uploadApplePushCertificate,
  uploadSiteIcon,
  uploadUserIcon,
} from "./api/dashboard.js";
import { fetchCurrentUser, logout } from "./api/session.js";
import { formatDeviceType } from "./lib/client.js";
import { makeDefaultAvatar } from "./lib/avatar.js";
import { byId, query } from "./lib/dom.js";
import { renderMarkdown } from "./lib/marked.js";
import { base64URLToBuffer, credentialToJSON } from "./lib/passkey.js";
import { hydrateSiteBrand, renderSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme, applyTheme, ThemeName } from "./lib/theme.js";
import type {
  ApplePushCertificate,
  BotPayload,
  BotUser,
  EntrySummary,
  LLMConfig,
  LLMConfigPayload,
  LoginRecord,
  SiteSettings,
  Tag,
  TagPayload,
} from "./types/dashboard.js";

const welcomeText = byId<HTMLElement>("welcomeText");
const entryList = byId<HTMLUListElement>("entryList");
const entryContent = byId<HTMLElement>("entryContent");
const logoutBtn = byId<HTMLButtonElement>("logoutBtn");
const newEntryBtn = byId<HTMLButtonElement>("newEntryBtn");
const loadMoreBtn = byId<HTMLButtonElement>("loadMoreBtn");
const editBtn = byId<HTMLButtonElement>("editBtn");
const deleteBtn = byId<HTMLButtonElement>("deleteBtn");
const drawerToggleBtn = byId<HTMLButtonElement>("drawerToggleBtn");
const drawerCloseBtn = byId<HTMLButtonElement>("drawerCloseBtn");
const drawerBackdrop = byId<HTMLElement>("drawerBackdrop");
const entryDrawer = byId<HTMLElement>("entryDrawer");
const loginHistoryList = byId<HTMLUListElement>("loginHistoryList");
const themeToggleBtn = byId<HTMLButtonElement>("themeToggleBtn");
const passkeyRegisterBtn = byId<HTMLButtonElement>("passkeyRegisterBtn");
const passkeyStatus = byId<HTMLElement>("passkeyStatus");
const userIcon = byId<HTMLImageElement>("userIcon");
const iconFile = byId<HTMLInputElement>("iconFile");
const iconEditor = byId<HTMLElement>("iconEditor");
const iconCanvas = byId<HTMLCanvasElement>("iconCanvas");
const iconZoom = byId<HTMLInputElement>("iconZoom");
const saveIconBtn = byId<HTMLButtonElement>("saveIconBtn");
const cancelIconBtn = byId<HTMLButtonElement>("cancelIconBtn");
const iconStatus = byId<HTMLElement>("iconStatus");
const settingsCardAvatar = byId<HTMLImageElement>("settingsCardAvatar");
const settingsCardName = byId<HTMLElement>("settingsCardName");
const settingsCardMeta = byId<HTMLElement>("settingsCardMeta");
const settingsProfileName = byId<HTMLElement>("settingsProfileName");
const settingsProfileMeta = byId<HTMLElement>("settingsProfileMeta");
const addTagBtn = byId<HTMLButtonElement>("addTagBtn");
const tagModal = byId<HTMLElement>("tagModal");
const tagModalTitle = byId<HTMLElement>("tagModalTitle");
const tagModalCloseBtn = byId<HTMLButtonElement>("tagModalCloseBtn");
const tagForm = byId<HTMLFormElement>("tagForm");
const tagName = byId<HTMLInputElement>("tagName");
const tagSlug = byId<HTMLInputElement>("tagSlug");
const tagDesc = byId<HTMLTextAreaElement>("tagDesc");
const tagOrder = byId<HTMLInputElement>("tagOrder");
const tagFormStatus = byId<HTMLElement>("tagFormStatus");
const tagSubmitBtn = byId<HTMLButtonElement>("tagSubmitBtn");
const openSiteAdminBtn = byId<HTMLButtonElement>("openSiteAdminBtn");
const siteAdminModal = byId<HTMLElement>("siteAdminModal");
const siteAdminModalTitle = byId<HTMLElement>("siteAdminModalTitle");
const siteAdminModalCloseBtn = byId<HTMLButtonElement>("siteAdminModalCloseBtn");
const settingsSectionLead = byId<HTMLElement>("settingsSectionLead");
const siteAdminPanel = byId<HTMLElement>("siteAdminPanel");
const themeCurrentValue = byId<HTMLElement>("themeCurrentValue");
const llmConfigForm = byId<HTMLFormElement>("llmConfigForm");
const llmConfigNameInput = byId<HTMLInputElement>("llmConfigNameInput");
const llmConfigBaseUrlInput = byId<HTMLInputElement>("llmConfigBaseUrlInput");
const llmConfigModelInput = byId<HTMLInputElement>("llmConfigModelInput");
const llmConfigApiKeyInput = byId<HTMLInputElement>("llmConfigApiKeyInput");
const llmConfigSystemPromptInput = byId<HTMLTextAreaElement>("llmConfigSystemPromptInput");
const llmConfigResetBtn = byId<HTMLButtonElement>("llmConfigResetBtn");
const llmConfigTestBtn = byId<HTMLButtonElement>("llmConfigTestBtn");
const llmConfigSubmitBtn = byId<HTMLButtonElement>("llmConfigSubmitBtn");
const llmConfigStatus = byId<HTMLElement>("llmConfigStatus");
const llmConfigList = byId<HTMLUListElement>("llmConfigList");
const botUserForm = byId<HTMLFormElement>("botUserForm");
const botUserNameInput = byId<HTMLInputElement>("botUserNameInput");
const botUserConfigSelect = byId<HTMLSelectElement>("botUserConfigSelect");
const botUserDescriptionInput = byId<HTMLTextAreaElement>("botUserDescriptionInput");
const botUserSystemPromptInput = byId<HTMLTextAreaElement>("botUserSystemPromptInput");
const botUserResetBtn = byId<HTMLButtonElement>("botUserResetBtn");
const botUserSubmitBtn = byId<HTMLButtonElement>("botUserSubmitBtn");
const botUserStatus = byId<HTMLElement>("botUserStatus");
const botUserList = byId<HTMLUListElement>("botUserList");
const siteNameInput = byId<HTMLInputElement>("siteNameInput");
const siteDescriptionInput = byId<HTMLTextAreaElement>("siteDescriptionInput");
const saveSiteBtn = byId<HTMLButtonElement>("saveSiteBtn");
const siteStatus = byId<HTMLElement>("siteStatus");
const siteIconPreview = byId<HTMLImageElement>("siteIconPreview");
const siteIconFile = byId<HTMLInputElement>("siteIconFile");
const applePushDevFile = byId<HTMLInputElement>("applePushDevFile");
const applePushProdFile = byId<HTMLInputElement>("applePushProdFile");
const applePushDevMeta = byId<HTMLElement>("applePushDevMeta");
const applePushProdMeta = byId<HTMLElement>("applePushProdMeta");
const applePushDevDeleteBtn = byId<HTMLButtonElement>("applePushDevDeleteBtn");
const applePushProdDeleteBtn = byId<HTMLButtonElement>("applePushProdDeleteBtn");
const siteAddTagBtnProxy = byId<HTMLButtonElement>("siteAddTagBtnProxy");
const tagList = byId<HTMLUListElement>("tagList");
const settingsNavButtons = Array.from(document.querySelectorAll<HTMLButtonElement>("[data-settings-nav]"));
const settingsPanels = Array.from(document.querySelectorAll<HTMLElement>("[data-settings-panel]"));
const settingsOpenButtons = Array.from(document.querySelectorAll<HTMLElement>("[data-open-settings-center], #openSiteAdminBtn"));

const iconCtx = iconCanvas.getContext("2d");

let nextOffset = 0;
let hasMore = true;
let activeEntryId: number | null = null;
let iconImage: HTMLImageElement | null = null;
let baseScale = 1;
let zoomValue = 1;
let offsetX = 0;
let offsetY = 0;
let dragging = false;
let dragStartX = 0;
let dragStartY = 0;
let isAdmin = false;
let editingTagId: number | null = null;
let currentTags: Tag[] = [];
let editingLLMConfigId: number | null = null;
let editingBotUserId: number | null = null;
let currentLLMConfigs: LLMConfig[] = [];
let currentBotUsers: BotUser[] = [];
let activeSettingsSection: "profile" | "personalization" | "settings" | "bots" | "site" = "personalization";

function setModalOpen(modal: HTMLElement, open: boolean): void {
  modal.classList.toggle("open", open);
  modal.setAttribute("aria-hidden", open ? "false" : "true");
}

function isMobileLayout(): boolean {
  return window.innerWidth <= 860;
}

function setDrawerOpen(open: boolean): void {
  if (!isMobileLayout()) {
    entryDrawer.classList.remove("open");
    drawerBackdrop.classList.remove("open");
    return;
  }
  entryDrawer.classList.toggle("open", open);
  drawerBackdrop.classList.toggle("open", open);
}

function setActiveEntryItem(): void {
  entryList.querySelectorAll<HTMLLIElement>("li[data-entry-id]").forEach((item) => {
    item.classList.toggle("active", Number(item.dataset.entryId) === activeEntryId);
  });
}

function syncThemeButton(theme: ThemeName): void {
  themeToggleBtn.textContent = theme === "mono" ? "切换到默认样式" : "切换到黑白样式";
  themeCurrentValue.textContent = theme === "mono" ? "黑白" : "默认";
}

function switchSettingsSection(section: "profile" | "personalization" | "settings" | "bots" | "site"): void {
  activeSettingsSection = section;
  const titles: Record<typeof activeSettingsSection, { title: string; lead: string }> = {
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
      lead: "管理 LLM Config，以及管理员可见的站点 logo、intro、iOS Push 证书和 Tag。",
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

function formatLocation(record: LoginRecord): string {
  const parts = [record.city, record.region, record.country].filter(Boolean);
  return parts.length > 0 ? parts.join(", ") : "位置未知";
}

function formatLoginMethod(method?: string): string {
  if (method === "passkey") {
    return "Passkey";
  }
  if (method === "register") {
    return "注册";
  }
  return "密码";
}

function defaultSiteIcon(name: string): string {
  return makeDefaultAvatar(name || "站", 160);
}

function formatCertificateMeta(cert?: ApplePushCertificate): string {
  if (!cert?.file_name) {
    return "未上传";
  }
  const uploadedAt = cert.uploaded_at ? new Date(cert.uploaded_at).toLocaleString() : "未知时间";
  return `当前文件：${cert.file_name} · 上传时间：${uploadedAt}`;
}

function renderSiteSettings(site?: SiteSettings): void {
  const safeSite = site || { name: "Polar-", description: "", icon_url: "" };
  siteNameInput.value = safeSite.name || "Polar-";
  siteDescriptionInput.value = safeSite.description || "";
  siteIconPreview.src = safeSite.icon_url || defaultSiteIcon(safeSite.name || "Polar-");
  applePushDevMeta.textContent = formatCertificateMeta(safeSite.apple_push_dev_cert);
  applePushProdMeta.textContent = formatCertificateMeta(safeSite.apple_push_prod_cert);
  applePushDevDeleteBtn.disabled = !safeSite.apple_push_dev_cert?.file_url;
  applePushProdDeleteBtn.disabled = !safeSite.apple_push_prod_cert?.file_url;
  renderSiteBrand(safeSite);
}

function renderTagList(tags: Tag[]): void {
  if (!tags.length) {
    tagList.innerHTML = `<li class="tag-item tag-item-empty">还没有 Tag，先创建第一个吧。</li>`;
    return;
  }

  tagList.innerHTML = tags
    .map(
      (tag) => `
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
      `
    )
    .join("");
}

function resetLLMConfigForm(): void {
  editingLLMConfigId = null;
  llmConfigForm.reset();
  llmConfigApiKeyInput.value = "";
  llmConfigStatus.textContent = "";
  llmConfigSubmitBtn.textContent = "保存配置";
}

function resetBotUserForm(): void {
  editingBotUserId = null;
  botUserForm.reset();
  botUserStatus.textContent = "";
  botUserSubmitBtn.textContent = "保存 Bot";
}

function renderLLMConfigList(configs: LLMConfig[]): void {
  if (!configs.length) {
    llmConfigList.innerHTML = `<li class="tag-item tag-item-empty">还没有 LLM Config，先创建一个吧。</li>`;
    return;
  }

  llmConfigList.innerHTML = configs
    .map(
      (config) => `
        <li class="tag-item" data-llm-config-id="${config.id}">
          <div class="tag-item-main">
            <div class="tag-item-header">
              <strong>${config.name}</strong>
              <span class="tag-chip">${config.model}</span>
              ${config.has_api_key ? `<span class="tag-chip">Key 已保存</span>` : `<span class="tag-chip">无 Key</span>`}
            </div>
            <div class="tag-item-meta">${config.base_url}</div>
            <div class="tag-item-desc">${config.system_prompt || "未设置 System Prompt"}</div>
          </div>
          <div class="tag-item-actions">
            <button class="btn-inline btn-secondary" type="button" data-action="edit">编辑</button>
            <button class="btn-inline" type="button" data-action="delete">删除</button>
          </div>
        </li>
      `
    )
    .join("");
}

function syncBotConfigOptions(configs: LLMConfig[]): void {
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

function renderBotUserList(bots: BotUser[]): void {
  if (!bots.length) {
    botUserList.innerHTML = `<li class="tag-item tag-item-empty">还没有 Bot，创建后就能直接私聊使用。</li>`;
    return;
  }

  botUserList.innerHTML = bots
    .map(
      (bot) => `
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
      `
    )
    .join("");
}

async function loadLoginHistory(): Promise<void> {
  const { response, data } = await fetchLoginHistory();
  if (!response.ok) {
    loginHistoryList.innerHTML = "<li>无法加载登录记录</li>";
    return;
  }

  const records: LoginRecord[] = data.records || [];
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

async function loadProfile(): Promise<void> {
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

async function loadSiteAdminData(): Promise<void> {
  const tasks: Array<Promise<unknown>> = [
    (async () => {
      const { response, data } = await fetchLLMConfigs();
      if (response.ok) {
        currentLLMConfigs = data.configs || [];
        renderLLMConfigList(currentLLMConfigs);
        syncBotConfigOptions(currentLLMConfigs);
      } else {
        llmConfigStatus.textContent = data.error || "无法加载 LLM Config";
        llmConfigList.innerHTML = `<li class="tag-item tag-item-empty">无法加载 LLM Config</li>`;
      }
    })(),
    (async () => {
      const { response, data } = await fetchBotUsers();
      if (response.ok) {
        currentBotUsers = data.bots || [];
        renderBotUserList(currentBotUsers);
      } else {
        botUserStatus.textContent = data.error || "无法加载 Bot 列表";
        botUserList.innerHTML = `<li class="tag-item tag-item-empty">无法加载 Bot 列表</li>`;
      }
    })(),
  ];

  if (isAdmin) {
    tasks.push(
      (async () => {
        const [siteResult, tagResult] = await Promise.all([fetchSiteSettings(), fetchTags()]);
        if (siteResult.response.ok) {
          renderSiteSettings(siteResult.data.site);
        } else {
          siteStatus.textContent = siteResult.data.error || "无法加载站点信息";
        }

        if (tagResult.response.ok) {
          currentTags = tagResult.data.tags || [];
          renderTagList(currentTags);
        } else {
          tagList.innerHTML = `<li class="tag-item tag-item-empty">无法加载 Tag 列表</li>`;
        }
      })()
    );
  }

  await Promise.all(tasks);
}

function openTagModal(tag?: Tag): void {
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

function closeTagModal(): void {
  editingTagId = null;
  setModalOpen(tagModal, false);
}

function openSiteAdminModal(section: "profile" | "personalization" | "settings" = "personalization"): void {
  switchSettingsSection(section);
  setModalOpen(siteAdminModal, true);
  if (section === "settings") {
    llmConfigNameInput.focus();
    return;
  }
  if (section === "profile") {
    passkeyRegisterBtn.focus();
    return;
  }
  themeToggleBtn.focus();
}

function closeSiteAdminModal(): void {
  setModalOpen(siteAdminModal, false);
}

async function loadEntries(reset = false): Promise<void> {
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
  const entries: EntrySummary[] = data.entries || [];
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

async function loadEntry(id: number): Promise<void> {
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

function drawIconPreview(): void {
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
  iconCtx.drawImage(
    iconImage,
    (iconCanvas.width - drawW) / 2 + offsetX,
    (iconCanvas.height - drawH) / 2 + offsetY,
    drawW,
    drawH
  );
}

function startDrag(clientX: number, clientY: number): void {
  dragging = true;
  dragStartX = clientX;
  dragStartY = clientY;
}

function moveDrag(clientX: number, clientY: number): void {
  if (!dragging) {
    return;
  }
  offsetX += clientX - dragStartX;
  offsetY += clientY - dragStartY;
  dragStartX = clientX;
  dragStartY = clientY;
  drawIconPreview();
}

function stopDrag(): void {
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
    const target = (button.dataset.settingsTarget as "profile" | "personalization" | "settings" | undefined) || "personalization";
    openSiteAdminModal(target);
  });
});
siteAdminModalCloseBtn.addEventListener("click", closeSiteAdminModal);
query<HTMLElement>(siteAdminModal, ".modal-backdrop").addEventListener("click", closeSiteAdminModal);
settingsNavButtons.forEach((button) => {
  button.addEventListener("click", () => {
    const target = button.dataset.settingsNav as "profile" | "personalization" | "settings" | undefined;
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
query<HTMLElement>(tagModal, ".modal-backdrop").addEventListener("click", closeTagModal);

tagForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  tagFormStatus.textContent = editingTagId ? "正在保存..." : "正在创建...";
  tagSubmitBtn.disabled = true;

  const payload: TagPayload = {
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
  } catch {
    tagFormStatus.textContent = "保存失败，请重试";
  } finally {
    tagSubmitBtn.disabled = false;
  }
});

tagList.addEventListener("click", async (event) => {
  const target = event.target as HTMLElement;
  const button = target.closest<HTMLButtonElement>("button[data-action]");
  if (!button) {
    return;
  }

  const item = button.closest<HTMLElement>("[data-tag-id]");
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
  const payload: LLMConfigPayload = {
    name: llmConfigNameInput.value.trim(),
    base_url: llmConfigBaseUrlInput.value.trim(),
    model: llmConfigModelInput.value.trim(),
    api_key: llmConfigApiKeyInput.value.trim(),
    system_prompt: llmConfigSystemPromptInput.value.trim(),
  };
  if (!payload.base_url || !payload.model || !payload.api_key) {
    llmConfigStatus.textContent = "测试前请先填写 Base URL、Model 和 API Key";
    return;
  }

  llmConfigTestBtn.disabled = true;
  llmConfigSubmitBtn.disabled = true;
  llmConfigStatus.textContent = "正在测试模型配置...";
  try {
    const { response, data } = await testLLMConfig(payload);
    llmConfigStatus.textContent = response.ok ? data.message || "连接成功，模型配置可用" : data.error || "测试失败";
  } catch {
    llmConfigStatus.textContent = "网络错误，请重试";
  } finally {
    llmConfigTestBtn.disabled = false;
    llmConfigSubmitBtn.disabled = false;
  }
});

botUserResetBtn.addEventListener("click", () => {
  resetBotUserForm();
});

llmConfigForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const payload: LLMConfigPayload = {
    name: llmConfigNameInput.value.trim(),
    base_url: llmConfigBaseUrlInput.value.trim(),
    model: llmConfigModelInput.value.trim(),
    api_key: llmConfigApiKeyInput.value.trim(),
    system_prompt: llmConfigSystemPromptInput.value.trim(),
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
  } catch {
    llmConfigStatus.textContent = "网络错误，请重试";
  } finally {
    llmConfigSubmitBtn.disabled = false;
  }
});

llmConfigList.addEventListener("click", async (event) => {
  const target = event.target as HTMLElement;
  const button = target.closest<HTMLButtonElement>("button[data-action]");
  if (!button) {
    return;
  }
  const item = button.closest<HTMLElement>("[data-llm-config-id]");
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
  const payload: BotPayload = {
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
  } catch {
    botUserStatus.textContent = "网络错误，请重试";
  } finally {
    botUserSubmitBtn.disabled = false;
  }
});

botUserList.addEventListener("click", async (event) => {
  const target = event.target as HTMLElement;
  const button = target.closest<HTMLButtonElement>("button[data-action]");
  if (!button) {
    return;
  }
  const item = button.closest<HTMLElement>("[data-bot-user-id]");
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
  } catch {
    siteStatus.textContent = "网络错误，请重试";
  } finally {
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
  } catch {
    siteStatus.textContent = "网络错误，请重试";
  } finally {
    siteIconFile.value = "";
  }
});

async function handleApplePushCertificateUpload(environment: "dev" | "prod", fileInput: HTMLInputElement): Promise<void> {
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
  } catch {
    siteStatus.textContent = "网络错误，请重试";
  } finally {
    fileInput.value = "";
  }
}

async function handleApplePushCertificateDelete(environment: "dev" | "prod"): Promise<void> {
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
  } catch {
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
  } catch {
    welcomeText.textContent = "退出失败，请检查网络后重试";
  } finally {
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
    } catch {
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
    passkeyStatus.textContent = "当前浏览器不支持 Passkey。";
    return;
  }

  passkeyStatus.textContent = "正在启动 Passkey...";
  try {
    const { response: beginResponse, data: beginResult } = await beginPasskeyRegistration();
    if (!beginResponse.ok) {
      passkeyStatus.textContent = beginResult.error || "无法发起 Passkey 绑定";
      return;
    }

    const publicKey = beginResult.publicKey as {
      challenge: string;
      user: { id: string };
      excludeCredentials?: Array<{ id: string; type: string }>;
    };
    publicKey.challenge = base64URLToBuffer(publicKey.challenge) as unknown as string;
    publicKey.user.id = base64URLToBuffer(publicKey.user.id) as unknown as string;
    if (publicKey.excludeCredentials) {
      publicKey.excludeCredentials = publicKey.excludeCredentials.map((cred) => ({
        ...cred,
        id: base64URLToBuffer(cred.id) as unknown as string,
      }));
    }

    const credential = await navigator.credentials.create({
      publicKey: publicKey as unknown as PublicKeyCredentialCreationOptions,
    });
    const payload = credentialToJSON(credential);

    const { response: finishResponse, data: finishResult } = await finishPasskeyRegistration(
      beginResult.session_id || "",
      payload
    );

    passkeyStatus.textContent = finishResponse.ok
      ? "Passkey 绑定成功！"
      : finishResult.error || "Passkey 绑定失败";
  } catch {
    passkeyStatus.textContent = "网络错误，请重试";
  }
});

const initialTheme = initStoredTheme();
syncThemeButton(initialTheme);
bindThemeSync(syncThemeButton);
switchSettingsSection(activeSettingsSection);

void (async () => {
  await hydrateSiteBrand();
  await loadProfile();
  await Promise.all([loadEntries(true), loadLoginHistory(), loadSiteAdminData()]);
})();
