import { byId } from "./lib/dom.js";
import { renderMarkdown } from "./lib/marked.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";

const API_BASE = "";
const alertBox = byId<HTMLElement>("alert");
const titleInput = byId<HTMLInputElement>("titleInput");
const contentInput = byId<HTMLTextAreaElement>("contentInput");
const preview = byId<HTMLElement>("preview");
const saveBtn = byId<HTMLButtonElement>("saveBtn");
const backBtn = byId<HTMLButtonElement>("backBtn");
const welcomeText = byId<HTMLElement>("welcomeText");
const entryId = new URLSearchParams(window.location.search).get("id");

initStoredTheme();
bindThemeSync();

async function ensureLogin(): Promise<void> {
  const res = await fetch(`${API_BASE}/api/me`, { credentials: "include" });
  if (!res.ok) {
    window.location.href = "/login.html";
  }
}

function renderPreview(): void {
  const raw = contentInput.value.trim();
  if (!raw) {
    preview.textContent = "暂无内容";
    return;
  }
  preview.innerHTML = renderMarkdown(raw);
}

async function loadEntry(): Promise<void> {
  if (!entryId) {
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
  renderPreview();
  welcomeText.textContent = "编辑记录";
}

contentInput.addEventListener("input", renderPreview);

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
      body: JSON.stringify({ title, content }),
    });
    const data = await res.json();

    if (!res.ok) {
      alertBox.className = "alert error";
      alertBox.textContent = data.error || "保存失败";
      return;
    }

    alertBox.className = "alert success";
    alertBox.textContent = entryId ? "更新成功" : `保存成功（ID: ${data.id}）`;
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
