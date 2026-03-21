import { buildAssetUrl, resolveAvatar } from "./lib/avatar.js";
import { byId, query } from "./lib/dom.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";
const API_BASE = "";
const postWelcome = byId("postWelcome");
const postList = byId("postList");
const postLoadMoreBtn = byId("postLoadMoreBtn");
let nextOffset = 0;
let hasMore = true;
let currentUserId = "";
let currentUserRole = "user";
let videoModal = null;
let videoModalPlayer = null;
initStoredTheme();
bindThemeSync();
function formatTime(value) {
    return new Date(value).toLocaleString();
}
function ensureVideoModal() {
    if (videoModal) {
        return;
    }
    const modal = document.createElement("div");
    modal.className = "video-modal";
    modal.innerHTML = `
    <div class="video-modal-backdrop"></div>
    <div class="video-modal-content panel">
      <button class="video-modal-close btn-inline btn-secondary" type="button">关闭</button>
      <video class="video-modal-player" controls autoplay preload="metadata"></video>
    </div>
  `;
    document.body.appendChild(modal);
    videoModal = modal;
    videoModalPlayer = query(modal, ".video-modal-player");
    const close = () => {
        if (!videoModal) {
            return;
        }
        videoModal.classList.remove("open");
        if (videoModalPlayer) {
            videoModalPlayer.pause();
            videoModalPlayer.removeAttribute("src");
            videoModalPlayer.load();
        }
    };
    query(modal, ".video-modal-backdrop").addEventListener("click", close);
    query(modal, ".video-modal-close").addEventListener("click", close);
    document.addEventListener("keydown", (event) => {
        if (event.key === "Escape") {
            close();
        }
    });
}
function openVideoModal(url) {
    if (!url) {
        return;
    }
    ensureVideoModal();
    if (!videoModal || !videoModalPlayer) {
        return;
    }
    videoModalPlayer.src = url;
    videoModal.classList.add("open");
    void videoModalPlayer.play().catch(() => { });
}
function normalizeVideoItems(post) {
    if (Array.isArray(post.video_items) && post.video_items.length > 0) {
        return post.video_items
            .filter((item) => item && item.url)
            .map((item) => ({
            url: buildAssetUrl(item.url),
            posterUrl: item.poster_url ? buildAssetUrl(item.poster_url) : "",
        }));
    }
    return (post.videos || []).map((url) => ({
        url: buildAssetUrl(url),
        posterUrl: "",
    }));
}
function enhancePostVideos(container) {
    container.querySelectorAll(".post-videos video").forEach((videoEl) => {
        videoEl.addEventListener("click", (event) => {
            event.preventDefault();
            event.stopPropagation();
            const source = videoEl.querySelector("source");
            const src = videoEl.currentSrc || source?.src || "";
            videoEl.pause();
            openVideoModal(src);
        });
    });
}
async function loadProfile() {
    const res = await fetch(`${API_BASE}/api/me`, { credentials: "include" });
    if (!res.ok) {
        window.location.href = "/login.html";
        return;
    }
    const data = await res.json();
    currentUserId = data.user_id;
    currentUserRole = data.role || "user";
    postWelcome.textContent = `你好，${data.username}，发布你的第一篇帖子吧。`;
}
function createPostCard(post) {
    const card = document.createElement("div");
    card.className = "post-card panel";
    const images = (post.images || [])
        .map((url) => `<img src="${buildAssetUrl(url)}" alt="post image" />`)
        .join("");
    const videos = normalizeVideoItems(post)
        .map((item) => `
        <video controls preload="metadata" ${item.posterUrl ? `poster="${item.posterUrl}"` : ""}>
          <source src="${item.url}" />
          你的浏览器不支持 video 标签
        </video>
      `)
        .join("");
    const videoSection = videos ? `<div class="post-videos">${videos}</div>` : "";
    const isSelf = currentUserId && post.user_id === currentUserId;
    const canDelete = currentUserRole === "admin" || isSelf;
    const authorLabel = isSelf
        ? `<span class="post-author-name">${post.username}</span>`
        : `<a class="post-author-name chat-link" href="/chat.html?user_id=${encodeURIComponent(post.user_id)}&username=${encodeURIComponent(post.username)}">${post.username}</a>`;
    const avatar = resolveAvatar(post.username, post.user_icon, 64);
    card.innerHTML = `
    <div class="post-header">
      <div class="post-author">
        <img class="avatar-sm" src="${avatar}" alt="${post.username}" />
        ${authorLabel}
      </div>
      <div class="post-time">${formatTime(post.created_at)}</div>
    </div>
    <div class="post-content">${post.content}</div>
    <div class="post-images">${images}</div>
    ${videoSection}
    <div class="post-actions">
      <button class="btn-inline btn-secondary like-btn" type="button">
        ${post.liked_by_me ? "已点赞" : "点赞"} · ${post.like_count}
      </button>
      <a class="btn-inline btn-secondary" href="/post.html?id=${post.id}">查看详情</a>
      ${canDelete ? '<button class="btn-inline btn-secondary delete-post-btn" type="button">删除帖子</button>' : ""}
    </div>
  `;
    const likeBtn = query(card, ".like-btn");
    const deleteBtn = card.querySelector(".delete-post-btn");
    enhancePostVideos(card);
    likeBtn.addEventListener("click", async () => {
        const method = post.liked_by_me ? "DELETE" : "POST";
        const res = await fetch(`${API_BASE}/api/posts/${post.id}/like`, {
            method,
            credentials: "include",
        });
        if (!res.ok) {
            return;
        }
        post.liked_by_me = !post.liked_by_me;
        post.like_count += post.liked_by_me ? 1 : -1;
        likeBtn.textContent = `${post.liked_by_me ? "已点赞" : "点赞"} · ${post.like_count}`;
    });
    deleteBtn?.addEventListener("click", async () => {
        if (!window.confirm("确认删除这条帖子吗？此操作不可恢复。")) {
            return;
        }
        deleteBtn.disabled = true;
        const res = await fetch(`${API_BASE}/api/posts/${post.id}`, {
            method: "DELETE",
            credentials: "include",
        });
        if (!res.ok) {
            deleteBtn.disabled = false;
            return;
        }
        card.remove();
        if (!postList.querySelector(".post-card")) {
            postList.innerHTML = "<div class='post-empty'>暂无帖子</div>";
        }
    });
    return card;
}
async function loadPosts(reset = false) {
    if (reset) {
        nextOffset = 0;
        hasMore = true;
        postList.innerHTML = "";
    }
    if (!hasMore) {
        return;
    }
    const res = await fetch(`${API_BASE}/api/posts?limit=10&offset=${nextOffset}`, {
        credentials: "include",
    });
    if (!res.ok) {
        postList.innerHTML = "<div class='post-empty'>无法加载帖子</div>";
        return;
    }
    const data = await res.json();
    const posts = data.posts || [];
    if (reset && posts.length === 0) {
        postList.innerHTML = "<div class='post-empty'>暂无帖子</div>";
        hasMore = false;
        postLoadMoreBtn.style.display = "none";
        return;
    }
    posts.forEach((post) => {
        postList.appendChild(createPostCard(post));
    });
    hasMore = Boolean(data.has_more);
    nextOffset = Number(data.next_offset || 0);
    postLoadMoreBtn.style.display = hasMore ? "inline-flex" : "none";
}
postLoadMoreBtn.addEventListener("click", () => {
    void loadPosts(false);
});
async function init() {
    await loadProfile();
    await loadPosts(true);
}
void init();
