import { resolveAvatar } from "./lib/avatar.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";
import { byId } from "./lib/dom.js";
import { t } from "./lib/i18n.js";
const markdownList = byId("markdownList");
const markdownLoadMoreBtn = byId("markdownLoadMoreBtn");
let nextOffset = 0;
let hasMore = true;
initStoredTheme();
bindThemeSync();
function formatTime(value) {
    return new Date(value).toLocaleString();
}
function createMarkdownCard(entry) {
    const card = document.createElement("a");
    card.className = "markdown-card panel";
    card.href = `/markdown.html?id=${entry.id}`;
    const avatar = resolveAvatar(entry.username, entry.user_icon, 64);
    card.innerHTML = `
    <div class="post-header">
      <div class="post-author">
        <img class="avatar-sm" src="${avatar}" alt="${entry.username}" />
        <span class="post-author-name">${entry.username}</span>
      </div>
      <div class="post-time">${formatTime(entry.uploaded_at)}</div>
    </div>
    <div class="markdown-card-title">${entry.title}</div>
    <div class="markdown-card-meta">${t("markdowns.clickToView")}</div>
  `;
    return card;
}
async function loadMarkdowns(reset = false) {
    if (reset) {
        nextOffset = 0;
        hasMore = true;
        markdownList.innerHTML = "";
    }
    if (!hasMore) {
        return;
    }
    const response = await fetch(`/api/public/markdowns?limit=10&offset=${nextOffset}`, {
        credentials: "include",
    });
    const data = await response.json();
    if (!response.ok) {
        markdownList.innerHTML = `<div class='post-empty'>${t("markdowns.loadFailed")}</div>`;
        return;
    }
    const entries = data.entries || [];
    if (reset && entries.length === 0) {
        markdownList.innerHTML = `<div class='post-empty'>${t("markdowns.noPosts")}</div>`;
        hasMore = false;
        markdownLoadMoreBtn.style.display = "none";
        return;
    }
    entries.forEach((entry) => {
        markdownList.appendChild(createMarkdownCard(entry));
    });
    hasMore = Boolean(data.has_more);
    nextOffset = Number(data.next_offset || 0);
    markdownLoadMoreBtn.style.display = hasMore ? "inline-flex" : "none";
}
markdownLoadMoreBtn.addEventListener("click", () => {
    void loadMarkdowns(false);
});
void hydrateSiteBrand();
void loadMarkdowns(true);
