import { beginPasskeyRegistration, createTag, deleteEntry, fetchEntries, fetchEntry, fetchLoginHistory, finishPasskeyRegistration, uploadUserIcon, } from "./api/dashboard.js";
import { fetchCurrentUser, logout } from "./api/session.js";
import { makeDefaultAvatar } from "./lib/avatar.js";
import { byId, query } from "./lib/dom.js";
import { renderMarkdown } from "./lib/marked.js";
import { base64URLToBuffer, credentialToJSON } from "./lib/passkey.js";
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
const tagModalCloseBtn = byId("tagModalCloseBtn");
const tagForm = byId("tagForm");
const tagName = byId("tagName");
const tagSlug = byId("tagSlug");
const tagDesc = byId("tagDesc");
const tagOrder = byId("tagOrder");
const tagFormStatus = byId("tagFormStatus");
const tagSubmitBtn = byId("tagSubmitBtn");
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
          <div class="meta-title">${record.ip_address || "未知 IP"} · ${formatLoginMethod(record.login_method)}</div>
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
    welcomeText.textContent = `你好，${data.username}`;
    const isAdmin = data.role === "admin";
    groupName.textContent = isAdmin ? "管理用户组" : "普通用户组";
    groupMeta.textContent = isAdmin ? "可管理标签与内容" : "基础浏览与发帖";
    addTagBtn.disabled = !isAdmin;
    addTagBtn.textContent = isAdmin ? "添加 Tag" : "仅管理员可添加";
    if (data.icon_url) {
        userIcon.src = data.icon_url;
    }
    else {
        userIcon.src = makeDefaultAvatar(data.username || "U", 160);
    }
}
function openTagModal() {
    tagForm.reset();
    tagOrder.value = "0";
    tagFormStatus.textContent = "";
    tagModal.classList.add("open");
    tagModal.setAttribute("aria-hidden", "false");
    tagName.focus();
}
function closeTagModal() {
    tagModal.classList.remove("open");
    tagModal.setAttribute("aria-hidden", "true");
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
        li.textContent = entry.title;
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
tagModalCloseBtn.addEventListener("click", closeTagModal);
query(tagModal, ".modal-backdrop").addEventListener("click", closeTagModal);
tagForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    tagFormStatus.textContent = "正在创建...";
    tagSubmitBtn.disabled = true;
    const payload = {
        name: tagName.value.trim(),
        slug: tagSlug.value.trim(),
        description: tagDesc.value.trim(),
        sort_order: Number(tagOrder.value || 0),
    };
    try {
        const { response, data } = await createTag(payload);
        if (!response.ok) {
            tagFormStatus.textContent = data.error || "创建失败";
            return;
        }
        tagFormStatus.textContent = "创建成功";
        window.setTimeout(() => {
            closeTagModal();
        }, 600);
    }
    catch {
        tagFormStatus.textContent = "创建失败，请重试";
    }
    finally {
        tagSubmitBtn.disabled = false;
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
void loadProfile();
void loadEntries(true);
void loadLoginHistory();
