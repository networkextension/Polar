import { byId } from "./lib/dom.js";
import { renderMarkdown } from "./lib/marked.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";

const API_BASE = "";
const alertBox = byId<HTMLElement>("alert");
const titleInput = byId<HTMLInputElement>("titleInput");
const contentInput = byId<HTMLTextAreaElement>("contentInput");
const preview = byId<HTMLElement>("preview");
const saveBtn = byId<HTMLButtonElement>("saveBtn");
const backBtn = byId<HTMLButtonElement>("backBtn");
const welcomeText = byId<HTMLElement>("welcomeText");
const publicToggle = byId<HTMLInputElement>("publicToggle");
const publicHint = byId<HTMLElement>("publicHint");
const entryId = new URLSearchParams(window.location.search).get("id");

let canEdit = true;

initStoredTheme();
bindThemeSync();

async function ensureLogin(): Promise<void> {
  const res = await fetch(`${API_BASE}/api/me`, { credentials: "include" });
  if (!res.ok) {
    window.location.href = "/login.html";
  }
}

function getPublicUrl(): string {
  if (!entryId) {
    return "";
  }
  return `${window.location.origin}/markdown.html?id=${encodeURIComponent(entryId)}`;
}

function updatePublicHint(): void {
  if (!canEdit) {
    publicHint.textContent = publicToggle.checked
      ? `当前是公开只读文档：${getPublicUrl()}`
      : "当前文档为只读，只有作者可以编辑。";
    return;
  }

  if (!publicToggle.checked) {
    publicHint.textContent = "默认仅自己可见。";
    return;
  }

  publicHint.textContent = entryId
    ? `其他用户可通过 ${getPublicUrl()} 查看此文档。`
    : "保存后会生成公开访问链接，其他用户可查看但不能编辑。";
}

function renderPreview(): void {
  const raw = contentInput.value.trim();
  if (!raw) {
    preview.textContent = "暂无内容";
    return;
  }
  preview.innerHTML = renderMarkdown(raw);
}

function applyReadonlyState(readonly: boolean): void {
  canEdit = !readonly;
  titleInput.disabled = readonly;
  contentInput.disabled = readonly;
  publicToggle.disabled = readonly;
  saveBtn.hidden = readonly;
  saveBtn.disabled = readonly;
  welcomeText.textContent = readonly ? "公开文档只读预览" : entryId ? "编辑记录" : "新建一条记录";
  updatePublicHint();
}

async function loadEntry(): Promise<void> {
  if (!entryId) {
    updatePublicHint();
    return;
  }

  const res = await fetch(`${API_BASE}/api/markdown/${entryId}`, {
    credentials: "include",
  });
  if (!res.ok) {
    alertBox.className = "alert error";
    alertBox.textContent = "无法加载记录";
    return;
  }

  const data = await res.json();
  titleInput.value = data.entry ? data.entry.title : "";
  contentInput.value = data.content || "";
  publicToggle.checked = Boolean(data.entry?.is_public);
  renderPreview();
  applyReadonlyState(data.can_edit === false);

  if (!canEdit) {
    alertBox.className = "alert success";
    alertBox.textContent = "你正在查看公开文档，只能阅读，不能编辑。";
  }
}

contentInput.addEventListener("input", renderPreview);
publicToggle.addEventListener("change", updatePublicHint);

saveBtn.addEventListener("click", async () => {
  alertBox.className = "alert";
  alertBox.textContent = "";

  const title = titleInput.value.trim();
  const content = contentInput.value.trim();
  if (!title || !content) {
    alertBox.className = "alert error";
    alertBox.textContent = "标题和内容不能为空";
    return;
  }

  try {
    const targetUrl = entryId ? `${API_BASE}/api/markdown/${entryId}` : `${API_BASE}/api/markdown`;
    const method = entryId ? "PUT" : "POST";
    const res = await fetch(targetUrl, {
      method,
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({
        title,
        content,
        is_public: publicToggle.checked,
      }),
    });
    const data = await res.json();

    if (!res.ok) {
      alertBox.className = "alert error";
      alertBox.textContent = data.error || "保存失败";
      return;
    }

    alertBox.className = "alert success";
    alertBox.textContent = entryId
      ? "更新成功"
      : `保存成功（ID: ${data.id}）`;
    window.location.href = "/dashboard.html";
  } catch {
    alertBox.className = "alert error";
    alertBox.textContent = "网络错误，请稍后重试";
  }
});

backBtn.addEventListener("click", () => {
  window.location.href = "/dashboard.html";
});

void ensureLogin();
void loadEntry();
void hydrateSiteBrand();
