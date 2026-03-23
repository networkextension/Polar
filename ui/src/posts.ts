import { buildAssetUrl, resolveAvatar } from "./lib/avatar.js";
import { byId, query } from "./lib/dom.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";
import { fetchTags } from "./api/dashboard.js";

const API_BASE = "";
const postWelcome = byId<HTMLElement>("postWelcome");
const postList = byId<HTMLElement>("postList");
const postLoadMoreBtn = byId<HTMLButtonElement>("postLoadMoreBtn");
const postTypeFilters = byId<HTMLElement>("postTypeFilters");
const tagFilters = byId<HTMLElement>("tagFilters");
const postListBadge = byId<HTMLElement>("postListBadge");

type VideoItem = {
  url: string;
  poster_url?: string;
};

type Reply = {
  username: string;
  user_icon?: string;
  created_at: string;
  content: string;
};

type Post = {
  id: number;
  user_id: string;
  username: string;
  user_icon?: string;
  tag_id?: number | null;
  post_type?: string;
  created_at: string;
  content: string;
  images?: string[];
  videos?: string[];
  video_items?: VideoItem[];
  liked_by_me: boolean;
  like_count: number;
  task?: {
    location?: string;
    start_at: string;
    end_at: string;
    working_hours: string;
    apply_deadline: string;
    application_status: string;
    applicant_count?: number;
  };
};

type Tag = {
  id: number;
  name: string;
  slug: string;
};

let nextOffset = 0;
let hasMore = true;
let currentUserId = "";
let currentUserRole = "user";
let videoModal: HTMLDivElement | null = null;
let videoModalPlayer: HTMLVideoElement | null = null;
let currentPostTypeFilter: "all" | "standard" | "task" = "all";
let currentTagFilter: number | null = null;
let currentTags: Tag[] = [];

initStoredTheme();
bindThemeSync();

function formatTime(value: string): string {
  return new Date(value).toLocaleString();
}

function profileUrl(userId: string): string {
  return `/profile.html?user_id=${encodeURIComponent(userId)}`;
}

function getTagName(tagId?: number | null): string {
  if (!tagId) {
    return "";
  }
  return currentTags.find((item) => item.id === tagId)?.name || "";
}

function updateListBadge(): void {
  if (currentTagFilter) {
    postListBadge.textContent = getTagName(currentTagFilter) || "板块帖子";
    return;
  }
  postListBadge.textContent =
    currentPostTypeFilter === "task" ? "零工任务" : currentPostTypeFilter === "standard" ? "普通帖子" : "最新帖子";
}

function renderTypeFilters(): void {
  const items: Array<{ label: string; value: "all" | "standard" | "task" }> = [
    { label: "最新帖子", value: "all" },
    { label: "普通帖子", value: "standard" },
    { label: "零工", value: "task" },
  ];
  postTypeFilters.innerHTML = items
    .map(
      (item) =>
        `<button class="btn-inline btn-secondary post-filter-btn ${currentPostTypeFilter === item.value && !currentTagFilter ? "active" : ""}" data-post-type="${item.value}" type="button">${item.label}</button>`
    )
    .join("");
  postTypeFilters.querySelectorAll<HTMLButtonElement>(".post-filter-btn").forEach((button) => {
    button.addEventListener("click", async () => {
      currentPostTypeFilter = (button.dataset.postType as "all" | "standard" | "task") || "all";
      currentTagFilter = null;
      renderTypeFilters();
      renderTagFilters();
      updateListBadge();
      await loadPosts(true);
    });
  });
}

function renderTagFilters(): void {
  tagFilters.innerHTML = currentTags
    .map(
      (tag) =>
        `<button class="tag-chip post-tag-filter ${currentTagFilter === tag.id ? "active" : ""}" data-tag-id="${tag.id}" type="button">${tag.name}</button>`
    )
    .join("");
  tagFilters.querySelectorAll<HTMLButtonElement>(".post-tag-filter").forEach((button) => {
    button.addEventListener("click", async () => {
      currentTagFilter = Number(button.dataset.tagId);
      currentPostTypeFilter = "all";
      renderTypeFilters();
      renderTagFilters();
      updateListBadge();
      await loadPosts(true);
    });
  });
}

async function loadTags(): Promise<void> {
  renderTypeFilters();
  updateListBadge();
  const { response, data } = await fetchTags();
  if (!response.ok) {
    return;
  }
  currentTags = data.tags || [];
  renderTypeFilters();
  renderTagFilters();
  updateListBadge();
}

function ensureVideoModal(): void {
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
  videoModalPlayer = query<HTMLVideoElement>(modal, ".video-modal-player");

  const close = (): void => {
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

  query<HTMLElement>(modal, ".video-modal-backdrop").addEventListener("click", close);
  query<HTMLButtonElement>(modal, ".video-modal-close").addEventListener("click", close);
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      close();
    }
  });
}

function openVideoModal(url: string): void {
  if (!url) {
    return;
  }
  ensureVideoModal();
  if (!videoModal || !videoModalPlayer) {
    return;
  }
  videoModalPlayer.src = url;
  videoModal.classList.add("open");
  void videoModalPlayer.play().catch(() => {});
}

function normalizeVideoItems(post: Post): Array<{ url: string; posterUrl: string }> {
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

function enhancePostVideos(container: ParentNode): void {
  container.querySelectorAll<HTMLVideoElement>(".post-videos video").forEach((videoEl) => {
    videoEl.addEventListener("click", (event) => {
      event.preventDefault();
      event.stopPropagation();
      const source = videoEl.querySelector<HTMLSourceElement>("source");
      const src = videoEl.currentSrc || source?.src || "";
      videoEl.pause();
      openVideoModal(src);
    });
  });
}

async function loadProfile(): Promise<void> {
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

function createPostCard(post: Post): HTMLElement {
  const card = document.createElement("div");
  card.className = "post-card panel";

  const images = (post.images || [])
    .map((url) => `<img src="${buildAssetUrl(url)}" alt="post image" />`)
    .join("");
  const videos = normalizeVideoItems(post)
    .map(
      (item) => `
        <video controls preload="metadata" ${item.posterUrl ? `poster="${item.posterUrl}"` : ""}>
          <source src="${item.url}" />
          你的浏览器不支持 video 标签
        </video>
      `
    )
    .join("");
  const videoSection = videos ? `<div class="post-videos">${videos}</div>` : "";

  const isSelf = currentUserId && post.user_id === currentUserId;
  const canDelete = currentUserRole === "admin" || isSelf;
  const authorLabel = `<a class="post-author-name" href="${profileUrl(post.user_id)}">${post.username}</a>`;

  const avatar = resolveAvatar(post.username, post.user_icon, 64);
  const taskSummary =
    post.post_type === "task" && post.task
      ? `
        <div class="task-summary-strip">
          <span class="badge">零工任务</span>
          <span>时间：${formatTime(post.task.start_at)} - ${formatTime(post.task.end_at)}</span>
          <span>Working hours：${post.task.working_hours}</span>
          <span>申请截止：${formatTime(post.task.apply_deadline)}</span>
          <span>申请数：${post.task.applicant_count || 0}</span>
        </div>
      `
      : "";
  const tagName = getTagName(post.tag_id);
  const tagSummary = tagName ? `<span class="tag-chip">${tagName}</span>` : "";

  card.innerHTML = `
    <div class="post-header">
      <div class="post-author">
        <a href="${profileUrl(post.user_id)}"><img class="avatar-sm" src="${avatar}" alt="${post.username}" /></a>
        ${authorLabel}
      </div>
      <div class="post-time">${tagSummary}${tagSummary ? " · " : ""}${formatTime(post.created_at)}</div>
    </div>
    <div class="post-content">${post.content}</div>
    ${taskSummary}
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

  const likeBtn = query<HTMLButtonElement>(card, ".like-btn");
  const deleteBtn = card.querySelector<HTMLButtonElement>(".delete-post-btn");
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

async function loadPosts(reset = false): Promise<void> {
  if (reset) {
    nextOffset = 0;
    hasMore = true;
    postList.innerHTML = "";
  }
  if (!hasMore) {
    return;
  }

  const params = new URLSearchParams({
    limit: "10",
    offset: String(nextOffset),
    post_type: currentTagFilter ? "all" : currentPostTypeFilter,
  });
  if (currentTagFilter) {
    params.set("tag_id", String(currentTagFilter));
  }
  const res = await fetch(`${API_BASE}/api/posts?${params.toString()}`, {
    credentials: "include",
  });
  if (!res.ok) {
    postList.innerHTML = "<div class='post-empty'>无法加载帖子</div>";
    return;
  }

  const data = await res.json();
  const posts: Post[] = data.posts || [];
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

async function init(): Promise<void> {
  await hydrateSiteBrand();
  await loadProfile();
  await loadTags();
  await loadPosts(true);
}

void init();
