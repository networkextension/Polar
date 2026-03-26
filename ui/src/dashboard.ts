import {
  createBotUser,
  createLLMConfig,
  beginPasskeyRegistration,
  createTag,
  deleteApplePushCertificate,
  deleteEntry,
  fetchAvailableLLMConfigs,
  fetchBotUsers,
  fetchEntries,
  fetchEntry,
  fetchLLMConfigs,
  fetchLoginHistory,
  fetchPasskeys,
  fetchSiteSettings,
  fetchTags,
  finishPasskeyRegistration,
  removePasskey,
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
import { t } from "./lib/i18n.js";
import type {
  ApplePushCertificate,
  BotPayload,
  BotUser,
  EntrySummary,
  LLMConfig,
  LLMConfigPayload,
  LoginRecord,
  PasskeyCredential,
  SiteSettings,
  SystemInfo,
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
const passkeyList = byId<HTMLUListElement>("passkeyList");
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
const siteManageCard = byId<HTMLElement>("siteManageCard");
const siteManageCardIcon = byId<HTMLImageElement>("siteManageCardIcon");
const siteManageCardName = byId<HTMLElement>("siteManageCardName");
const siteManageCardMeta = byId<HTMLElement>("siteManageCardMeta");
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
const llmConfigSharedInput = byId<HTMLInputElement>("llmConfigSharedInput");
const llmConfigResetBtn = byId<HTMLButtonElement>("llmConfigResetBtn");
const llmConfigTestBtn = byId<HTMLButtonElement>("llmConfigTestBtn");
const llmConfigSubmitBtn = byId<HTMLButtonElement>("llmConfigSubmitBtn");
const llmConfigStatus = byId<HTMLElement>("llmConfigStatus");
const llmConfigList = byId<HTMLUListElement>("llmConfigList");
const settingsGitTagVersion = byId<HTMLElement>("settingsGitTagVersion");
const settingsOS = byId<HTMLElement>("settingsOS");
const settingsCPUArch = byId<HTMLElement>("settingsCPUArch");
const settingsPartitionCapacity = byId<HTMLElement>("settingsPartitionCapacity");
const settingsPartitionPath = byId<HTMLElement>("settingsPartitionPath");
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
let currentAvailableLLMConfigs: LLMConfig[] = [];
let currentBotUsers: BotUser[] = [];
let activeSettingsSection: "profile" | "personalization" | "settings" | "system" | "bots" | "site" = "personalization";

function setStatusMessage(element: HTMLElement, message: string, tone: "default" | "success" | "error" = "default"): void {
  element.textContent = message;
  element.classList.remove("status-success", "status-error");
  if (tone === "success") {
    element.classList.add("status-success");
  } else if (tone === "error") {
    element.classList.add("status-error");
  }
}

function setModalOpen(modal: HTMLElement, open: boolean): void {
  modal.classList.toggle("open", open);
  modal.setAttribute("aria-hidden", open ? "false" : "true");
  if (modal === siteAdminModal || modal === tagModal) {
    document.body.classList.toggle("modal-open", open);
  }
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
  themeToggleBtn.textContent = theme === "mono" ? t("dashboard.switchToDefault") : t("dashboard.switchToMonochrome");
  themeCurrentValue.textContent = theme === "mono" ? t("dashboard.themeMonochrome") : t("dashboard.themeDefault");
}

function switchSettingsSection(section: "profile" | "personalization" | "settings" | "system" | "bots" | "site"): void {
  activeSettingsSection = section;
  const titles: Record<typeof activeSettingsSection, { title: string; lead: string }> = {
    profile: {
      title: t("dashboard.personalCenter"),
      lead: t("dashboard.profileLead"),
    },
    personalization: {
      title: t("dashboard.personalizationTitle"),
      lead: t("dashboard.personalizationLead"),
    },
    settings: {
      title: t("dashboard.settingsTitle"),
      lead: "",
    },
    system: {
      title: "系统信息",
      lead: "查看当前实例的版本、系统环境与程序所在分区的剩余容量。",
    },
    bots: {
      title: t("dashboard.botsManagementTitle"),
      lead: t("dashboard.botsLead"),
    },
    site: {
      title: t("dashboard.siteManagementTitle"),
      lead: t("dashboard.siteLead"),
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
  return parts.length > 0 ? parts.join(", ") : t("dashboard.unknownLocation");
}

function formatLoginMethod(method?: string): string {
  if (method === "passkey") {
    return "Passkey";
  }
  if (method === "register") {
    return t("dashboard.loginMethodRegister");
  }
  return t("dashboard.loginMethodPassword");
}

function formatPasskeyLabel(credentialId: string): string {
  if (!credentialId) {
    return "Passkey";
  }
  if (credentialId.length <= 12) {
    return credentialId;
  }
  return `${credentialId.slice(0, 6)}...${credentialId.slice(-6)}`;
}

function renderPasskeys(credentials: PasskeyCredential[] = []): void {
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

  passkeyList.querySelectorAll<HTMLButtonElement>("[data-passkey-delete]").forEach((button) => {
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
      } catch {
        setStatusMessage(passkeyStatus, "网络错误，请重试", "error");
        button.disabled = false;
      }
    });
  });
}

function defaultSiteIcon(name: string): string {
  return makeDefaultAvatar(name || "站", 160);
}

function formatCertificateMeta(cert?: ApplePushCertificate): string {
  if (!cert?.file_name) {
    return t("dashboard.notUploaded");
  }
  const uploadedAt = cert.uploaded_at ? new Date(cert.uploaded_at).toLocaleString() : t("dashboard.unknownTime");
  return t("dashboard.certMeta", { filename: cert.file_name, time: uploadedAt });
}

function renderSystemInfo(systemInfo?: SystemInfo): void {
  settingsGitTagVersion.textContent = systemInfo?.git_tag_version || "未知";
  settingsOS.textContent = systemInfo?.os || "未知";
  settingsCPUArch.textContent = systemInfo?.cpu_arch || "未知";
  settingsPartitionCapacity.textContent = systemInfo?.partition_capacity || "未知";
  settingsPartitionPath.textContent = systemInfo?.partition_path
    ? `路径：${systemInfo.partition_path}`
    : "";
}

function renderSiteSettings(site?: SiteSettings): void {
  const safeSite: SiteSettings = site || { name: "Polar-", description: "", icon_url: "" };
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

function renderTagList(tags: Tag[]): void {
  if (!tags.length) {
    tagList.innerHTML = `<li class="tag-item tag-item-empty">${t("dashboard.noTags")}</li>`;
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
            <div class="tag-item-meta">${t("dashboard.tagOrder", { order: String(tag.sort_order) })}</div>
            <div class="tag-item-desc">${tag.description || t("dashboard.noDescription")}</div>
          </div>
          <div class="tag-item-actions">
            <button class="btn-inline btn-secondary" type="button" data-action="edit">${t("common.edit")}</button>
            <button class="btn-inline" type="button" data-action="delete">${t("common.delete")}</button>
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
  llmConfigSubmitBtn.textContent = t("dashboard.saveConfigBtn");
}

function resetBotUserForm(): void {
  editingBotUserId = null;
  botUserForm.reset();
  botUserStatus.textContent = "";
  botUserSubmitBtn.textContent = t("dashboard.saveBotBtn");
}

function renderLLMConfigList(configs: LLMConfig[]): void {
  if (!configs.length) {
    llmConfigList.innerHTML = `<li class="tag-item tag-item-empty">${t("dashboard.noLLMConfigs")}</li>`;
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
              ${config.has_api_key ? `<span class="tag-chip">${t("dashboard.keySaved")}</span>` : `<span class="tag-chip">${t("dashboard.noKey")}</span>`}
              ${config.shared ? `<span class="tag-chip">${t("dashboard.shared")}</span>` : `<span class="tag-chip">${t("dashboard.private")}</span>`}
            </div>
            <div class="tag-item-meta">${config.base_url}</div>
            <div class="tag-item-desc">${config.system_prompt || t("dashboard.noSystemPrompt")}</div>
          </div>
          <div class="tag-item-actions">
            <button class="btn-inline btn-secondary" type="button" data-action="edit">${t("common.edit")}</button>
            <button class="btn-inline" type="button" data-action="delete">${t("common.delete")}</button>
          </div>
        </li>
      `
    )
    .join("");
}

function syncBotConfigOptions(configs: LLMConfig[]): void {
  if (!configs.length) {
    botUserConfigSelect.innerHTML = `<option value="">${t("dashboard.createLLMConfigFirst")}</option>`;
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
    botUserList.innerHTML = `<li class="tag-item tag-item-empty">${t("dashboard.noBots")}</li>`;
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
            <div class="tag-item-meta">${t("dashboard.botUserId", { id: bot.bot_user_id })}</div>
            <div class="tag-item-desc">${bot.description || t("dashboard.noDescription")}</div>
            <div class="tag-item-meta">${bot.system_prompt ? t("dashboard.botPromptPreview", { preview: bot.system_prompt.slice(0, 36) + (bot.system_prompt.length > 36 ? "" : "") }) : t("dashboard.noBotPrompt")}</div>
          </div>
          <div class="tag-item-actions">
            <button class="btn-inline btn-secondary" type="button" data-action="chat">${t("dashboard.chat")}</button>
            <button class="btn-inline btn-secondary" type="button" data-action="edit">${t("common.edit")}</button>
            <button class="btn-inline" type="button" data-action="delete">${t("common.delete")}</button>
          </div>
        </li>
      `
    )
    .join("");
}

async function loadLoginHistory(): Promise<void> {
  const { response, data } = await fetchLoginHistory();
  if (!response.ok) {
    loginHistoryList.innerHTML = `<li>${t("dashboard.loginHistoryLoadFailed")}</li>`;
    return;
  }

  const records: LoginRecord[] = data.records || [];
  if (!records.length) {
    loginHistoryList.innerHTML = `<li>${t("dashboard.noLoginHistory")}</li>`;
    return;
  }

  loginHistoryList.innerHTML = records
    .map((record) => {
      const time = new Date(record.logged_in_at).toLocaleString();
      return `
        <li>
          <div class="meta-title">${record.ip_address || t("dashboard.unknownIp")} · ${formatLoginMethod(record.login_method)} · ${formatDeviceType(record.device_type, t)}</div>
          <div class="meta-subtitle">${formatLocation(record)}</div>
          <div class="meta-time">${time}</div>
        </li>
      `;
    })
    .join("");
}

async function loadPasskeys(): Promise<void> {
  const { response, data } = await fetchPasskeys();
  if (!response.ok) {
    setStatusMessage(passkeyStatus, data.error || "无法加载 Passkey 状态", "error");
    passkeyList.innerHTML = "<li>无法加载 Passkey 列表</li>";
    return;
  }
  renderPasskeys(data.credentials || []);
}

async function loadProfile(): Promise<void> {
  const { response, data } = await fetchCurrentUser();
  if (!response.ok) {
    window.location.href = "/login.html";
    return;
  }

  isAdmin = data.role === "admin";
  welcomeText.textContent = t("dashboard.welcome", { username: data.username });
  settingsCardName.textContent = data.username || t("dashboard.currentUser");
  settingsCardMeta.textContent = isAdmin ? t("dashboard.adminMeta") : t("dashboard.userMeta");
  settingsProfileName.textContent = data.username || t("dashboard.currentUser");
  settingsProfileMeta.textContent = isAdmin ? t("dashboard.adminProfileMeta") : t("dashboard.userProfileMeta");
  addTagBtn.disabled = !isAdmin;
  addTagBtn.textContent = isAdmin ? t("dashboard.newTag") : t("dashboard.adminOnlyTag");
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

async function loadSiteAdminData(): Promise<void> {
  const tasks: Array<Promise<unknown>> = [
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
      } else {
        llmConfigStatus.textContent = ownResult.data.error || t("dashboard.llmConfigLoadFailed");
        llmConfigList.innerHTML = `<li class="tag-item tag-item-empty">${t("dashboard.llmConfigLoadFailed")}</li>`;
      }

      if (availableResult.response.ok) {
        currentAvailableLLMConfigs = availableResult.data.configs || [];
        syncBotConfigOptions(currentAvailableLLMConfigs);
      } else {
        currentAvailableLLMConfigs = [];
        botUserStatus.textContent = availableResult.data.error || t("dashboard.availableConfigLoadFailed");
        syncBotConfigOptions(currentAvailableLLMConfigs);
      }
    })(),
    (async () => {
      const { response, data } = await fetchBotUsers();
      if (response.ok) {
        currentBotUsers = data.bots || [];
        renderBotUserList(currentBotUsers);
      } else {
        botUserStatus.textContent = data.error || t("dashboard.botListLoadFailed");
        botUserList.innerHTML = `<li class="tag-item tag-item-empty">${t("dashboard.botListLoadFailed")}</li>`;
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
          siteStatus.textContent = siteResult.data.error || t("dashboard.siteInfoLoadFailed");
        }

        if (tagResult.response.ok) {
          currentTags = tagResult.data.tags || [];
        const { response, data } = await fetchTags();
        if (response.ok) {
          currentTags = data.tags || [];
          renderTagList(currentTags);
        } else {
          tagList.innerHTML = `<li class="tag-item tag-item-empty">${t("dashboard.tagListLoadFailed")}</li>`;
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
  tagModalTitle.textContent = editingTagId ? t("dashboard.editTag") : t("dashboard.addTag");
  tagSubmitBtn.textContent = editingTagId ? t("common.save") : t("common.create");
  setModalOpen(tagModal, true);
  tagName.focus();
}

function closeTagModal(): void {
  editingTagId = null;
  setModalOpen(tagModal, false);
}

function openSiteAdminModal(section: "profile" | "personalization" | "settings" | "system" = "personalization"): void {
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
    entryList.innerHTML = `<li>${t("dashboard.entryLoadFailed")}</li>`;
    return;
  }
  const entries: EntrySummary[] = data.entries || [];
  if (reset && !entries.length) {
    entryList.innerHTML = `<li>${t("dashboard.noEntries")}</li>`;
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
    entryContent.textContent = t("dashboard.entryReadFailed");
    return;
  }

  activeEntryId = data.entry ? data.entry.id : null;
  setActiveEntryItem();
  const rawContent = data.content || t("dashboard.emptyContent");
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
    const target = (button.dataset.settingsTarget as "profile" | "personalization" | "settings" | "system" | undefined) || "personalization";
    openSiteAdminModal(target);
  });
});
siteAdminModalCloseBtn.addEventListener("click", closeSiteAdminModal);
query<HTMLElement>(siteAdminModal, ".modal-backdrop").addEventListener("click", closeSiteAdminModal);
settingsNavButtons.forEach((button) => {
  button.addEventListener("click", () => {
    const target = button.dataset.settingsNav as
      | "profile"
      | "personalization"
      | "settings"
      | "system"
      | "bots"
      | "site"
      | undefined;
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
  tagFormStatus.textContent = editingTagId ? t("dashboard.tagSaving") : t("dashboard.tagCreating");
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
      tagFormStatus.textContent = data.error || t("dashboard.tagSaveFailed");
      return;
    }
    tagFormStatus.textContent = editingTagId ? t("dashboard.tagSaved") : t("dashboard.tagCreated");
    await loadSiteAdminData();
    window.setTimeout(() => {
      closeTagModal();
    }, 400);
  } catch {
    tagFormStatus.textContent = t("dashboard.tagSaveFailedRetry");
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
    if (!window.confirm(t("dashboard.confirmDeleteTag", { name: tag.name }))) {
      return;
    }

    const { response, data } = await removeTag(tag.id);
    if (!response.ok) {
      siteStatus.textContent = data.error || t("common.deleteFailed");
      return;
    }
    siteStatus.textContent = t("dashboard.tagDeleted", { name: tag.name });
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
    shared: llmConfigSharedInput.checked,
  };
  if (!payload.base_url || !payload.model || !payload.api_key) {
    setStatusMessage(llmConfigStatus, t("dashboard.llmTestMissingFields"), "error");
    return;
  }

  llmConfigTestBtn.disabled = true;
  llmConfigSubmitBtn.disabled = true;
  setStatusMessage(llmConfigStatus, t("dashboard.llmTesting"));
  try {
    const { response, data } = await testLLMConfig(payload);
    setStatusMessage(
      llmConfigStatus,
      response.ok ? data.message || t("dashboard.llmTestSuccess") : data.error || t("dashboard.llmTestFailed"),
      response.ok ? "success" : "error"
    );
  } catch {
    setStatusMessage(llmConfigStatus, t("common.networkErrorRetry"), "error");
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
    shared: llmConfigSharedInput.checked,
  };
  if (!payload.name || !payload.base_url || !payload.model) {
    llmConfigStatus.textContent = t("dashboard.llmConfigMissingFields");
    return;
  }

  llmConfigSubmitBtn.disabled = true;
  llmConfigStatus.textContent = editingLLMConfigId ? t("dashboard.llmConfigSaving") : t("dashboard.llmConfigCreating");
  try {
    const { response, data } = editingLLMConfigId
      ? await updateLLMConfig(editingLLMConfigId, {
          ...payload,
          update_api_key: payload.api_key !== "",
        })
      : await createLLMConfig(payload);
    if (!response.ok) {
      llmConfigStatus.textContent = data.error || t("common.saveFailed");
      return;
    }
    llmConfigStatus.textContent = editingLLMConfigId ? t("dashboard.llmConfigUpdated") : t("dashboard.llmConfigCreated");
    resetLLMConfigForm();
    await loadSiteAdminData();
  } catch {
    llmConfigStatus.textContent = t("common.networkErrorRetry");
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
    llmConfigSharedInput.checked = config.shared;
    llmConfigSubmitBtn.textContent = t("dashboard.updateConfig");
    llmConfigStatus.textContent = config.has_api_key ? t("dashboard.apiKeySaved") : "";
    llmConfigNameInput.focus();
    return;
  }

  if (action === "delete") {
    if (!window.confirm(t("dashboard.confirmDeleteConfig", { name: config.name }))) {
      return;
    }
    const { response, data } = await removeLLMConfig(config.id);
    if (!response.ok) {
      llmConfigStatus.textContent = data.error || t("common.deleteFailed");
      return;
    }
    llmConfigStatus.textContent = t("dashboard.configDeleted", { name: config.name });
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
    botUserStatus.textContent = t("dashboard.botMissingFields");
    return;
  }

  botUserSubmitBtn.disabled = true;
  botUserStatus.textContent = editingBotUserId ? t("dashboard.botSaving") : t("dashboard.botCreating");
  try {
    const { response, data } = editingBotUserId
      ? await updateBotUser(editingBotUserId, payload)
      : await createBotUser(payload);
    if (!response.ok) {
      botUserStatus.textContent = data.error || t("common.saveFailed");
      return;
    }
    botUserStatus.textContent = editingBotUserId ? t("dashboard.botUpdated") : t("dashboard.botCreated");
    resetBotUserForm();
    await loadSiteAdminData();
  } catch {
    botUserStatus.textContent = t("common.networkErrorRetry");
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
    botUserSubmitBtn.textContent = t("dashboard.updateBot");
    botUserStatus.textContent = "";
    botUserNameInput.focus();
    return;
  }
  if (action === "delete") {
    if (!window.confirm(t("dashboard.confirmDeleteBot", { name: bot.name }))) {
      return;
    }
    const { response, data } = await removeBotUser(bot.id);
    if (!response.ok) {
      botUserStatus.textContent = data.error || t("common.deleteFailed");
      return;
    }
    botUserStatus.textContent = t("dashboard.botDeleted", { name: bot.name });
    resetBotUserForm();
    await loadSiteAdminData();
  }
});

saveSiteBtn.addEventListener("click", async () => {
  siteStatus.textContent = t("dashboard.savingSiteInfo");
  saveSiteBtn.disabled = true;

  try {
    const { response, data } = await updateSiteSettings({
      name: siteNameInput.value.trim(),
      description: siteDescriptionInput.value.trim(),
      icon_url: siteIconPreview.src,
    });
    if (!response.ok) {
      siteStatus.textContent = data.error || t("common.saveFailed");
      return;
    }
    renderSiteSettings(data.site);
    siteStatus.textContent = t("dashboard.siteInfoSaved");
  } catch {
    siteStatus.textContent = t("common.networkErrorRetry");
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
    siteStatus.textContent = t("dashboard.selectImageFile");
    siteIconFile.value = "";
    return;
  }

  siteStatus.textContent = t("dashboard.uploadingSiteIcon");
  const formData = new FormData();
  formData.append("icon", file);

  try {
    const { response, data } = await uploadSiteIcon(formData);
    if (!response.ok) {
      siteStatus.textContent = data.error || t("common.submitFailed");
      return;
    }
    renderSiteSettings(data.site);
    siteIconPreview.src = `${data.icon_url || data.site?.icon_url || ""}?v=${Date.now()}`;
    siteStatus.textContent = t("dashboard.siteIconUpdated");
  } catch {
    siteStatus.textContent = t("common.networkErrorRetry");
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
    siteStatus.textContent = t("dashboard.certUnsupportedFormat");
    fileInput.value = "";
    return;
  }

  siteStatus.textContent = t("dashboard.uploadingCert", { env: environment });
  const formData = new FormData();
  formData.append("certificate", file);

  try {
    const { response, data } = await uploadApplePushCertificate(environment, formData);
    if (!response.ok) {
      siteStatus.textContent = data.error || t("common.submitFailed");
      return;
    }
    renderSiteSettings(data.site);
    siteStatus.textContent = t("dashboard.certUpdated", { env: environment });
  } catch {
    siteStatus.textContent = t("common.networkErrorRetry");
  } finally {
    fileInput.value = "";
  }
}

async function handleApplePushCertificateDelete(environment: "dev" | "prod"): Promise<void> {
  if (!window.confirm(t("dashboard.confirmDeleteCert", { env: environment }))) {
    return;
  }

  siteStatus.textContent = t("dashboard.deletingCert", { env: environment });
  try {
    const { response, data } = await deleteApplePushCertificate(environment);
    if (!response.ok) {
      siteStatus.textContent = data.error || t("common.deleteFailed");
      return;
    }
    renderSiteSettings(data.site);
    siteStatus.textContent = t("dashboard.certDeleted", { env: environment });
  } catch {
    siteStatus.textContent = t("common.networkErrorRetry");
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
      welcomeText.textContent = t("dashboard.logoutFailed");
      return;
    }
    window.location.replace("/login.html");
  } catch {
    welcomeText.textContent = t("dashboard.logoutNetworkFailed");
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
  if (!window.confirm(t("dashboard.confirmDeleteEntry"))) {
    return;
  }

  const response = await deleteEntry(activeEntryId);
  if (response.ok) {
    activeEntryId = null;
    entryContent.textContent = t("dashboard.selectEntry");
    await loadEntries(true);
  }
});

iconFile.addEventListener("change", () => {
  const file = iconFile.files?.[0];
  if (!file) {
    return;
  }
  if (!file.type.startsWith("image/")) {
    iconStatus.textContent = t("dashboard.selectImageFileWithPeriod");
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
  iconStatus.textContent = t("dashboard.canceledEdit");
});

saveIconBtn.addEventListener("click", async () => {
  if (!iconImage) {
    return;
  }
  iconStatus.textContent = t("common.uploading");
  iconCanvas.toBlob(async (blob) => {
    if (!blob) {
      iconStatus.textContent = t("dashboard.generatingImageFailed");
      return;
    }

    const formData = new FormData();
    formData.append("icon", blob, "icon.png");

    try {
      const { response, data } = await uploadUserIcon(formData);
      if (!response.ok) {
        iconStatus.textContent = data.error || t("common.submitFailed");
        return;
      }
      const nextAvatar = `${data.icon_url || ""}?v=${Date.now()}`;
      userIcon.src = nextAvatar;
      settingsCardAvatar.src = nextAvatar;
      iconEditor.classList.remove("active");
      iconFile.value = "";
      iconStatus.textContent = t("dashboard.avatarUpdated");
    } catch {
      iconStatus.textContent = t("dashboard.networkErrorPeriod");
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
    passkeyStatus.textContent = t("dashboard.passkeyNotSupported");
    return;
  }

  passkeyStatus.textContent = t("dashboard.passkeyStarting");
  try {
    const { response: beginResponse, data: beginResult } = await beginPasskeyRegistration();
    if (!beginResponse.ok) {
      passkeyStatus.textContent = beginResult.error || t("dashboard.passkeyBeginFailed");
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
      ? t("dashboard.passkeySuccess")
      : finishResult.error || t("dashboard.passkeyFailed");
  } catch {
    passkeyStatus.textContent = t("common.networkErrorRetry");
    if (!finishResponse.ok) {
      setStatusMessage(passkeyStatus, finishResult.error || "Passkey 绑定失败", "error");
      return;
    }
    renderPasskeys(finishResult.credentials || []);
    setStatusMessage(passkeyStatus, finishResult.message || "Passkey 绑定成功！", "success");
  } catch {
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
