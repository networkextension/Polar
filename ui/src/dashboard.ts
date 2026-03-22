import {
  beginPasskeyRegistration,
  createTag,
  deleteEntry,
  fetchEntries,
  fetchEntry,
  fetchLoginHistory,
  fetchSiteSettings,
  fetchTags,
  finishPasskeyRegistration,
  removeTag,
  updateSiteSettings,
  updateTag,
  uploadSiteIcon,
  uploadUserIcon,
} from "./api/dashboard.js";
import { fetchCurrentUser, logout } from "./api/session.js";
import { makeDefaultAvatar } from "./lib/avatar.js";
import { byId, query } from "./lib/dom.js";
import { renderMarkdown } from "./lib/marked.js";
import { base64URLToBuffer, credentialToJSON } from "./lib/passkey.js";
import { hydrateSiteBrand, renderSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme, applyTheme, ThemeName } from "./lib/theme.js";
import type {
  EntrySummary,
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
const groupName = byId<HTMLElement>("groupName");
const groupMeta = byId<HTMLElement>("groupMeta");
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
const siteAdminPanel = byId<HTMLElement>("siteAdminPanel");
const siteNameInput = byId<HTMLInputElement>("siteNameInput");
const siteDescriptionInput = byId<HTMLTextAreaElement>("siteDescriptionInput");
const saveSiteBtn = byId<HTMLButtonElement>("saveSiteBtn");
const siteStatus = byId<HTMLElement>("siteStatus");
const siteIconPreview = byId<HTMLImageElement>("siteIconPreview");
const siteIconFile = byId<HTMLInputElement>("siteIconFile");
const siteAddTagBtnProxy = byId<HTMLButtonElement>("siteAddTagBtnProxy");
const tagList = byId<HTMLUListElement>("tagList");

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

function renderSiteSettings(site?: SiteSettings): void {
  const safeSite = site || { name: "Polar-", description: "", icon_url: "" };
  siteNameInput.value = safeSite.name || "Polar-";
  siteDescriptionInput.value = safeSite.description || "";
  siteIconPreview.src = safeSite.icon_url || defaultSiteIcon(safeSite.name || "Polar-");
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
          <div class="meta-title">${record.ip_address || "未知 IP"} · ${formatLoginMethod(record.login_method)}</div>
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
  groupName.textContent = isAdmin ? "管理用户组" : "普通用户组";
  groupMeta.textContent = isAdmin ? "可管理站点信息、Tag 与内容" : "基础浏览与发帖";
  addTagBtn.disabled = !isAdmin;
  addTagBtn.textContent = isAdmin ? "新建 Tag" : "仅管理员可新建 Tag";
  addTagBtn.hidden = !isAdmin;
  siteAdminPanel.hidden = !isAdmin;

  if (data.icon_url) {
    userIcon.src = data.icon_url;
  } else {
    userIcon.src = makeDefaultAvatar(data.username || "U", 160);
  }
}

async function loadSiteAdminData(): Promise<void> {
  if (!isAdmin) {
    return;
  }

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
  tagModal.classList.add("open");
  tagModal.setAttribute("aria-hidden", "false");
  tagName.focus();
}

function closeTagModal(): void {
  editingTagId = null;
  tagModal.classList.remove("open");
  tagModal.setAttribute("aria-hidden", "true");
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

logoutBtn.addEventListener("click", async () => {
  await logout();
  window.location.href = "/login.html";
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
      userIcon.src = `${data.icon_url || ""}?v=${Date.now()}`;
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

void (async () => {
  await hydrateSiteBrand();
  await loadProfile();
  await Promise.all([loadEntries(true), loadLoginHistory(), loadSiteAdminData()]);
})();
