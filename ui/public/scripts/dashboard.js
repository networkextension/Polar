import { beginPasskeyRegistration, createTag, deleteApplePushCertificate, deleteEntry, fetchEntries, fetchEntry, fetchLoginHistory, fetchSiteSettings, fetchTags, finishPasskeyRegistration, removeTag, updateSiteSettings, updateTag, uploadApplePushCertificate, uploadSiteIcon, uploadUserIcon, } from "./api/dashboard.js";
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
const userIcon = byId("userIcon");
const iconFile = byId("iconFile");
const iconEditor = byId("iconEditor");
const iconCanvas = byId("iconCanvas");
const iconZoom = byId("iconZoom");
const saveIconBtn = byId("saveIconBtn");
const cancelIconBtn = byId("cancelIconBtn");
const iconStatus = byId("iconStatus");
const groupName = byId("groupName");
const groupMeta = byId("groupMeta");
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
const siteAdminModalCloseBtn = byId("siteAdminModalCloseBtn");
const siteAdminPanel = byId("siteAdminPanel");
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
function setModalOpen(modal, open) {
    modal.classList.toggle("open", open);
    modal.setAttribute("aria-hidden", open ? "false" : "true");
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
function renderSiteSettings(site) {
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
async function loadProfile() {
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
    openSiteAdminBtn.hidden = !isAdmin;
    siteAdminPanel.hidden = !isAdmin;
    if (data.icon_url) {
        userIcon.src = data.icon_url;
    }
    else {
        userIcon.src = makeDefaultAvatar(data.username || "U", 160);
    }
}
async function loadSiteAdminData() {
    if (!isAdmin) {
        return;
    }
    const [siteResult, tagResult] = await Promise.all([fetchSiteSettings(), fetchTags()]);
    if (siteResult.response.ok) {
        renderSiteSettings(siteResult.data.site);
    }
    else {
        siteStatus.textContent = siteResult.data.error || "无法加载站点信息";
    }
    if (tagResult.response.ok) {
        currentTags = tagResult.data.tags || [];
        renderTagList(currentTags);
    }
    else {
        tagList.innerHTML = `<li class="tag-item tag-item-empty">无法加载 Tag 列表</li>`;
    }
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
function openSiteAdminModal() {
    if (!isAdmin) {
        return;
    }
    setModalOpen(siteAdminModal, true);
    siteNameInput.focus();
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
openSiteAdminBtn.addEventListener("click", openSiteAdminModal);
siteAdminModalCloseBtn.addEventListener("click", closeSiteAdminModal);
query(siteAdminModal, ".modal-backdrop").addEventListener("click", closeSiteAdminModal);
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
            userIcon.src = `${data.icon_url || ""}?v=${Date.now()}`;
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
        passkeyStatus.textContent = finishResponse.ok
            ? "Passkey 绑定成功！"
            : finishResult.error || "Passkey 绑定失败";
    }
    catch {
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
