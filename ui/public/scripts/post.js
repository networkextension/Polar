import { buildAssetUrl, resolveAvatar } from "./lib/avatar.js";
import { byId, query } from "./lib/dom.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";
import { fetchTags } from "./api/dashboard.js";
const API_BASE = "";
const postWelcome = byId("postWelcome");
const postDetail = byId("postDetail");
const postForm = byId("postForm");
const postContent = byId("postContent");
const postType = byId("postType");
const postTag = byId("postTag");
const taskFields = byId("taskFields");
const taskLocation = byId("taskLocation");
const taskStartAt = byId("taskStartAt");
const taskEndAt = byId("taskEndAt");
const workingHours = byId("workingHours");
const applyDeadline = byId("applyDeadline");
const postImages = byId("postImages");
const postVideos = byId("postVideos");
const postFormStatus = byId("postFormStatus");
const postSubmitBtn = byId("postSubmitBtn");
let currentUserId = "";
let currentUserRole = "user";
let videoModal = null;
let videoModalPlayer = null;
let currentTags = [];
initStoredTheme();
bindThemeSync();
function getPostId() {
    return new URLSearchParams(window.location.search).get("id");
}
function formatTime(value) {
    return new Date(value).toLocaleString();
}
function toISOStringFromLocal(value) {
    return new Date(value).toISOString();
}
function escapeHtml(value) {
    return value
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}
function profileUrl(userId) {
    return `/profile.html?user_id=${encodeURIComponent(userId)}`;
}
function getTagName(tagId) {
    if (!tagId) {
        return "";
    }
    return currentTags.find((item) => item.id === tagId)?.name || "";
}
async function loadTagOptions() {
    const { response, data } = await fetchTags();
    if (!response.ok) {
        return;
    }
    currentTags = data.tags || [];
    postTag.innerHTML = [
        '<option value="">不选择板块</option>',
        ...currentTags.map((tag) => `<option value="${tag.id}">${tag.name}</option>`),
    ].join("");
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
    postWelcome.textContent = `你好，${data.username}`;
}
async function loadReplies(postId) {
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
    const replies = data.replies || [];
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
function renderPost(post) {
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
    const isTask = post.post_type === "task" && post.task;
    const authorLabel = `<a class="post-author-name" href="${profileUrl(post.user_id)}">${post.username}</a>`;
    const avatar = resolveAvatar(post.username, post.user_icon, 64);
    const taskInfo = isTask
        ? `
      <div class="task-meta-card">
        <div class="badge">零工任务</div>
        <div class="task-meta-grid">
          <div><strong>时间范围：</strong>${formatTime(post.task.start_at)} - ${formatTime(post.task.end_at)}</div>
          <div><strong>Working hours：</strong>${escapeHtml(post.task.working_hours)}</div>
          <div><strong>申请截止：</strong>${formatTime(post.task.apply_deadline)}</div>
          <div><strong>地理位置：</strong>${escapeHtml(post.task.location || "未限制")}</div>
          <div><strong>申请状态：</strong>${post.task.application_status === "open" ? "开放中" : "已关闭"}</div>
          <div><strong>当前申请数：</strong>${post.task.applicant_count || 0}</div>
          ${post.task.selected_applicant_name
            ? `<div><strong>已选候选人：</strong>${escapeHtml(post.task.selected_applicant_name)}</div>`
            : ""}
        </div>
      </div>
    `
        : "";
    const tagName = getTagName(post.tag_id);
    const tagInfo = tagName ? `<div class="tag-chip">${tagName}</div>` : "";
    const taskActions = isTask
        ? `
      <div class="task-actions">
        ${post.task.can_apply
            ? `<button id="taskApplyBtn" class="btn-inline btn-secondary" type="button">${post.task.applied_by_me ? "撤销申请" : "申请任务"}</button>`
            : ""}
        ${post.task.can_manage && post.task.application_status === "open"
            ? `<button id="taskCloseBtn" class="btn-inline btn-secondary" type="button">关闭申请</button>
               <button id="taskLoadApplicantsBtn" class="btn-inline btn-secondary" type="button">查看申请者</button>`
            : ""}
      </div>
      <div id="taskApplicantsPanel" class="task-applicants-panel"></div>
    `
        : "";
    const taskResultSection = isTask && post.task.can_view_results
        ? `
      <div class="task-results-card">
        <div class="badge">任务成果</div>
        ${post.task.can_submit_result
            ? `
              <form id="taskResultForm" class="task-result-form">
                <label class="form-label" for="taskResultNote">成果说明</label>
                <textarea id="taskResultNote" class="input textarea" rows="3" placeholder="补充说明本次完成内容..."></textarea>
                <label class="form-label" for="taskResultImages">成果图片</label>
                <input id="taskResultImages" class="input" type="file" accept="image/*" multiple />
                <label class="form-label" for="taskResultVideos">成果视频</label>
                <input id="taskResultVideos" class="input" type="file" accept="video/*" multiple />
                <div id="taskResultStatus" class="status-text"></div>
                <div class="task-form-actions">
                  <button id="taskResultSubmitBtn" class="btn-inline btn-secondary" type="submit">提交任务成果</button>
                </div>
              </form>
            `
            : ""}
        <div id="taskResultList" class="task-result-list"></div>
      </div>
    `
        : "";
    postDetail.innerHTML = `
    <div class="post-header">
      <div class="post-author">
        <a href="${profileUrl(post.user_id)}"><img class="avatar-sm" src="${avatar}" alt="${post.username}" /></a>
        ${authorLabel}
      </div>
      <div class="post-time">${formatTime(post.created_at)}</div>
    </div>
    <div class="post-content">${post.content}</div>
    ${tagInfo}
    ${taskInfo}
    <div class="post-images">${images}</div>
    ${videoSection}
    <div class="post-actions">
      <button id="detailLikeBtn" class="btn-inline btn-secondary" type="button">
        ${post.liked_by_me ? "已点赞" : "点赞"} · ${post.like_count}
      </button>
      ${canDelete ? '<button id="detailDeleteBtn" class="btn-inline btn-secondary" type="button">删除帖子</button>' : ""}
    </div>
    ${taskActions}
    ${taskResultSection}
    <div class="reply-box open">
      <div class="reply-list" id="replyList"></div>
      <form id="replyForm" class="reply-form">
        <input id="replyInput" class="input reply-input" type="text" placeholder="写下你的回复..." required />
        <button class="btn-inline btn-secondary" type="submit">发送</button>
      </form>
    </div>
  `;
    const likeBtn = byId("detailLikeBtn");
    const deleteBtn = document.getElementById("detailDeleteBtn");
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
    const replyForm = byId("replyForm");
    const replyInput = byId("replyInput");
    const taskApplyBtn = document.getElementById("taskApplyBtn");
    const taskCloseBtn = document.getElementById("taskCloseBtn");
    const taskLoadApplicantsBtn = document.getElementById("taskLoadApplicantsBtn");
    const taskApplicantsPanel = document.getElementById("taskApplicantsPanel");
    const taskResultForm = document.getElementById("taskResultForm");
    const taskResultNote = document.getElementById("taskResultNote");
    const taskResultImages = document.getElementById("taskResultImages");
    const taskResultVideos = document.getElementById("taskResultVideos");
    const taskResultStatus = document.getElementById("taskResultStatus");
    const taskResultSubmitBtn = document.getElementById("taskResultSubmitBtn");
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
    taskApplyBtn?.addEventListener("click", async () => {
        const method = post.task?.applied_by_me ? "DELETE" : "POST";
        const res = await fetch(`${API_BASE}/api/tasks/${post.id}/apply`, {
            method,
            credentials: "include",
        });
        if (!res.ok) {
            return;
        }
        await loadPost();
    });
    taskCloseBtn?.addEventListener("click", async () => {
        const res = await fetch(`${API_BASE}/api/tasks/${post.id}/close`, {
            method: "POST",
            credentials: "include",
        });
        if (!res.ok) {
            return;
        }
        await loadPost();
    });
    taskLoadApplicantsBtn?.addEventListener("click", async () => {
        if (!taskApplicantsPanel) {
            return;
        }
        taskApplicantsPanel.innerHTML = "<div class='reply-empty'>加载申请者中...</div>";
        const res = await fetch(`${API_BASE}/api/tasks/${post.id}/applications`, {
            credentials: "include",
        });
        if (!res.ok) {
            taskApplicantsPanel.innerHTML = "<div class='reply-empty'>无法加载申请者</div>";
            return;
        }
        const data = await res.json();
        const applications = data.applications || [];
        if (!applications.length) {
            taskApplicantsPanel.innerHTML = "<div class='reply-empty'>暂无申请者</div>";
            return;
        }
        taskApplicantsPanel.innerHTML = applications
            .map((item) => {
            const avatarUrl = resolveAvatar(item.username, item.user_icon, 40);
            const defaultTemplate = post.task?.invitation_template
                || `你好，你已被选为该零工任务的候选人。\n\n任务内容：${post.content}\n如果你确认参与，请直接回复。`;
            return `
          <div class="task-applicant-item">
            <div class="task-applicant-head">
              <a href="${profileUrl(item.user_id)}"><img class="avatar-xs" src="${avatarUrl}" alt="${item.username}" /></a>
              <div>
                <div class="reply-meta"><a class="post-author-name" href="${profileUrl(item.user_id)}">${item.username}</a> · ${formatTime(item.applied_at)}</div>
                <div class="task-applicant-id">${item.user_id}</div>
              </div>
            </div>
            <textarea class="input textarea task-template-input" data-user-id="${item.user_id}" rows="4">${escapeHtml(defaultTemplate)}</textarea>
            <button class="btn-inline btn-secondary task-select-btn" data-user-id="${item.user_id}" type="button">确认并发送私信</button>
          </div>
        `;
        })
            .join("");
        taskApplicantsPanel.querySelectorAll(".task-select-btn").forEach((button) => {
            button.addEventListener("click", async () => {
                const applicantUserId = button.dataset.userId;
                if (!applicantUserId) {
                    return;
                }
                const textarea = taskApplicantsPanel.querySelector(`.task-template-input[data-user-id="${applicantUserId}"]`);
                const messageTemplate = textarea?.value.trim() || "";
                button.disabled = true;
                const selectRes = await fetch(`${API_BASE}/api/tasks/${post.id}/select-candidate`, {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    credentials: "include",
                    body: JSON.stringify({
                        applicant_user_id: applicantUserId,
                        message_template: messageTemplate,
                    }),
                });
                if (!selectRes.ok) {
                    button.disabled = false;
                    return;
                }
                await loadPost();
            });
        });
    });
    taskResultForm?.addEventListener("submit", async (event) => {
        event.preventDefault();
        if (!taskResultStatus || !taskResultSubmitBtn || !taskResultImages || !taskResultVideos) {
            return;
        }
        const formData = new FormData();
        const note = taskResultNote?.value.trim() || "";
        if (note) {
            formData.append("note", note);
        }
        Array.from(taskResultImages.files || []).forEach((file) => formData.append("images", file));
        Array.from(taskResultVideos.files || []).forEach((file) => formData.append("videos", file));
        taskResultStatus.textContent = "正在提交任务成果...";
        taskResultSubmitBtn.disabled = true;
        const res = await fetch(`${API_BASE}/api/tasks/${post.id}/results`, {
            method: "POST",
            credentials: "include",
            body: formData,
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) {
            taskResultStatus.textContent = data.error || "提交失败";
            taskResultSubmitBtn.disabled = false;
            return;
        }
        taskResultStatus.textContent = "任务成果已提交";
        taskResultForm.reset();
        taskResultSubmitBtn.disabled = false;
        await loadTaskResults(post.id);
    });
    if (post.task?.can_view_results) {
        void loadTaskResults(post.id);
    }
}
async function loadTaskResults(postId) {
    const container = document.getElementById("taskResultList");
    if (!container) {
        return;
    }
    container.innerHTML = "<div class='reply-empty'>加载任务成果中...</div>";
    const res = await fetch(`${API_BASE}/api/tasks/${postId}/results`, {
        credentials: "include",
    });
    if (!res.ok) {
        container.innerHTML = "<div class='reply-empty'>无法加载任务成果</div>";
        return;
    }
    const data = await res.json();
    const results = data.results || [];
    if (!results.length) {
        container.innerHTML = "<div class='reply-empty'>暂未提交任务成果</div>";
        return;
    }
    container.innerHTML = results
        .map((result) => {
        const avatar = resolveAvatar(result.username, result.user_icon, 40);
        const images = (result.images || [])
            .map((url) => `<img src="${buildAssetUrl(url)}" alt="task result image" />`)
            .join("");
        const videos = (result.video_items || [])
            .map((item) => {
            const videoUrl = buildAssetUrl(item.url);
            const posterUrl = item.poster_url ? buildAssetUrl(item.poster_url) : "";
            return `
            <video controls preload="metadata" ${posterUrl ? `poster="${posterUrl}"` : ""}>
              <source src="${videoUrl}" />
              你的浏览器不支持 video 标签
            </video>
          `;
        })
            .join("");
        return `
        <div class="task-result-item">
          <div class="task-applicant-head">
            <a href="${profileUrl(result.user_id)}"><img class="avatar-xs" src="${avatar}" alt="${result.username}" /></a>
            <div class="reply-meta"><a class="post-author-name" href="${profileUrl(result.user_id)}">${result.username}</a> · ${formatTime(result.created_at)}</div>
          </div>
          ${result.note ? `<div class="post-content">${escapeHtml(result.note)}</div>` : ""}
          ${images ? `<div class="post-images">${images}</div>` : ""}
          ${videos ? `<div class="post-videos">${videos}</div>` : ""}
        </div>
      `;
    })
        .join("");
    enhancePostVideos(container);
}
async function loadPost() {
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
    const post = data.post || null;
    if (!post) {
        postDetail.innerHTML = "<div class='post-empty'>未找到帖子</div>";
        return;
    }
    renderPost(post);
    await loadReplies(post.id);
}
function syncTaskFields() {
    const enabled = postType.value === "task";
    taskFields.hidden = !enabled;
    taskStartAt.required = enabled;
    taskEndAt.required = enabled;
    workingHours.required = enabled;
    applyDeadline.required = enabled;
}
postType.addEventListener("change", syncTaskFields);
postForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const content = postContent.value.trim();
    if (!content) {
        postFormStatus.textContent = "内容不能为空";
        return;
    }
    if (postType.value === "task") {
        if (!taskStartAt.value || !taskEndAt.value || !workingHours.value.trim() || !applyDeadline.value) {
            postFormStatus.textContent = "请填写完整的任务信息";
            return;
        }
    }
    postFormStatus.textContent = "正在发布...";
    postSubmitBtn.disabled = true;
    const formData = new FormData();
    formData.append("post_type", postType.value);
    if (postTag.value) {
        formData.append("tag_id", postTag.value);
    }
    formData.append("content", content);
    if (postType.value === "task") {
        formData.append("task_location", taskLocation.value.trim());
        formData.append("task_start_at", toISOStringFromLocal(taskStartAt.value));
        formData.append("task_end_at", toISOStringFromLocal(taskEndAt.value));
        formData.append("working_hours", workingHours.value.trim());
        formData.append("apply_deadline", toISOStringFromLocal(applyDeadline.value));
    }
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
    }
    catch {
        postFormStatus.textContent = "发布失败，请重试";
    }
    finally {
        postSubmitBtn.disabled = false;
    }
});
async function init() {
    await hydrateSiteBrand();
    await loadProfile();
    await loadTagOptions();
    syncTaskFields();
    await loadPost();
}
void init();
