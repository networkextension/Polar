import { byId } from "./lib/dom.js";
import { renderMarkdown } from "./lib/marked.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";

const titleEl = byId<HTMLElement>("markdownTitle");
const metaEl = byId<HTMLElement>("markdownMeta");
const alertBox = byId<HTMLElement>("markdownAlert");
const contentEl = byId<HTMLElement>("markdownContent");
const entryId = new URLSearchParams(window.location.search).get("id");

initStoredTheme();
bindThemeSync();

function applyMarkdownPayload(data: { entry?: { title?: string; is_public?: boolean }; content?: string }): void {
  titleEl.textContent = data.entry?.title || "公开 Markdown";
  metaEl.textContent = data.entry?.is_public ? "公开只读文档" : "只读预览";
  contentEl.innerHTML = renderMarkdown(data.content || "");
}

async function requestMarkdown(path: string): Promise<{ ok: boolean; status: number; data: any }> {
  const response = await fetch(path, { credentials: "include" });
  let data: any = {};
  try {
    data = await response.json();
  } catch {
    data = {};
  }
  return {
    ok: response.ok,
    status: response.status,
    data,
  };
}

async function loadPublicMarkdown(): Promise<void> {
  if (!entryId) {
    alertBox.className = "alert error";
    alertBox.textContent = "缺少文档 ID";
    contentEl.textContent = "无法加载内容";
    return;
  }

  try {
    const publicResult = await requestMarkdown(`/api/public/markdown/${encodeURIComponent(entryId)}`);
    if (publicResult.ok) {
      applyMarkdownPayload(publicResult.data);
      return;
    }

    if (publicResult.status === 404) {
      const authResult = await requestMarkdown(`/api/markdown/${encodeURIComponent(entryId)}`);
      if (authResult.ok) {
        applyMarkdownPayload(authResult.data);
        metaEl.textContent = "登录态只读预览";
        return;
      }
    }

    alertBox.className = "alert error";
    alertBox.textContent = publicResult.data.error || "无法加载文档";
    contentEl.textContent = "未找到公开文档";
  } catch {
    alertBox.className = "alert error";
    alertBox.textContent = "网络错误，请稍后重试";
    contentEl.textContent = "加载失败";
  }
}

void hydrateSiteBrand();
void loadPublicMarkdown();
