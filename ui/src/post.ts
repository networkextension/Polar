import { buildAssetUrl, resolveAvatar } from "./lib/avatar.js";
import { byId, query } from "./lib/dom.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";

const API_BASE = "";
const postWelcome = byId<HTMLElement>("postWelcome");
const postDetail = byId<HTMLElement>("postDetail");
const postForm = byId<HTMLFormElement>("postForm");
const postContent = byId<HTMLTextAreaElement>("postContent");
const postImages = byId<HTMLInputElement>("postImages");
const postVideos = byId<HTMLInputElement>("postVideos");
const postFormStatus = byId<HTMLElement>("postFormStatus");
const postSubmitBtn = byId<HTMLButtonElement>("postSubmitBtn");

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
  created_at: string;
  content: string;
  images?: string[];
  videos?: string[];
  video_items?: VideoItem[];
  liked_by_me: boolean;
  like_count: number;
};

let currentUserId = "";
let currentUserRole = "user";
let videoModal: HTMLDivElement | null = null;
let videoModalPlayer: HTMLVideoElement | null = null;

initStoredTheme();
bindThemeSync();

function getPostId(): string | null {
  return new URLSearchParams(window.location.search).get("id");
}

function formatTime(value: string): string {
  return new Date(value).toLocaleString();
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
  postWelcome.textContent = `你好，${data.username}`;
}

async function loadReplies(postId: number): Promise<void> {
  const replyList = document.getElementById("replyList");
  if (!replyList) {
    return;
  }

  replyList.innerHTML = "<div class='reply-empty'>加载中...</div>";
  const res = await fetch(`${API_BASE}/api/posts/${postId}/replies?limit=50`, {
    credentials: "include",
  });
  if (!res.ok) {
    replyList.innerHTML = "<div class='reply-empty'>无法加载回复</div>";
    return;
  }

  const data = await res.json();
  const replies: Reply[] = data.replies || [];
  if (replies.length === 0) {
    replyList.innerHTML = "<div class='reply-empty'>暂无回复</div>";
    return;
  }

  replyList.innerHTML = replies
    .map((reply) => {
      const avatar = resolveAvatar(reply.username, reply.user_icon, 48);
      return `
        <div class="reply-item">
          <img class="avatar-xs" src="${avatar}" alt="${reply.username}" />
          <div class="reply-body">
            <div class="reply-meta">${reply.username} · ${formatTime(reply.created_at)}</div>
            <div class="reply-content">${reply.content}</div>
          </div>
        </div>
      `;
    })
    .join("");
}

function renderPost(post: Post): void {
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
  const authorLabel = isSelf
    ? `<span class="post-author-name">${post.username}</span>`
    : `<a class="post-author-name chat-link" href="/chat.html?user_id=${encodeURIComponent(
        post.user_id
      )}&username=${encodeURIComponent(post.username)}">${post.username}</a>`;

  const avatar = resolveAvatar(post.username, post.user_icon, 64);

  postDetail.innerHTML = `
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
      <button id="detailLikeBtn" class="btn-inline btn-secondary" type="button">
        ${post.liked_by_me ? "已点赞" : "点赞"} · ${post.like_count}
      </button>
      ${canDelete ? '<button id="detailDeleteBtn" class="btn-inline btn-secondary" type="button">删除帖子</button>' : ""}
    </div>
    <div class="reply-box open">
      <div class="reply-list" id="replyList"></div>
      <form id="replyForm" class="reply-form">
        <input id="replyInput" class="input reply-input" type="text" placeholder="写下你的回复..." required />
        <button class="btn-inline btn-secondary" type="submit">发送</button>
      </form>
    </div>
  `;

  const likeBtn = byId<HTMLButtonElement>("detailLikeBtn");
  const deleteBtn = document.getElementById("detailDeleteBtn") as HTMLButtonElement | null;
  enhancePostVideos(postDetail);

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
    window.location.href = "/posts.html";
  });

  const replyForm = byId<HTMLFormElement>("replyForm");
  const replyInput = byId<HTMLInputElement>("replyInput");

  replyForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const content = replyInput.value.trim();
    if (!content) {
      return;
    }
    const res = await fetch(`${API_BASE}/api/posts/${post.id}/replies`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ content }),
    });
    if (!res.ok) {
      return;
    }
    replyInput.value = "";
    await loadReplies(post.id);
  });
}

async function loadPost(): Promise<void> {
  const postId = getPostId();
  if (!postId) {
    postDetail.innerHTML = "<div class='post-empty'>无效的帖子</div>";
    return;
  }

  const res = await fetch(`${API_BASE}/api/posts/${postId}`, {
    credentials: "include",
  });
  if (!res.ok) {
    postDetail.innerHTML = "<div class='post-empty'>无法加载帖子</div>";
    return;
  }

  const data = await res.json();
  const post: Post | null = data.post || null;
  if (!post) {
    postDetail.innerHTML = "<div class='post-empty'>未找到帖子</div>";
    return;
  }

  renderPost(post);
  await loadReplies(post.id);
}

postForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const content = postContent.value.trim();
  if (!content) {
    postFormStatus.textContent = "内容不能为空";
    return;
  }

  postFormStatus.textContent = "正在发布...";
  postSubmitBtn.disabled = true;

  const formData = new FormData();
  formData.append("content", content);
  Array.from(postImages.files || []).forEach((file) => {
    formData.append("images", file);
  });
  Array.from(postVideos.files || []).forEach((file) => {
    formData.append("videos", file);
  });

  try {
    const res = await fetch(`${API_BASE}/api/posts`, {
      method: "POST",
      credentials: "include",
      body: formData,
    });
    const data = await res.json();
    if (!res.ok) {
      postFormStatus.textContent = data.error || "发布失败";
      return;
    }
    postFormStatus.textContent = "发布成功";
    postForm.reset();
    window.location.href = `/post.html?id=${data.id}`;
  } catch {
    postFormStatus.textContent = "发布失败，请重试";
  } finally {
    postSubmitBtn.disabled = false;
  }
});

async function init(): Promise<void> {
  await hydrateSiteBrand();
  await loadProfile();
  await loadPost();
}

void init();
