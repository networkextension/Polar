import { upsertRecommendation, updateMyProfile, fetchUserProfile } from "./api/profile.js";
import { uploadUserIcon } from "./api/dashboard.js";
import { fetchCurrentUser } from "./api/session.js";
import { resolveAvatar } from "./lib/avatar.js";
import { byId } from "./lib/dom.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { bindThemeSync, initStoredTheme } from "./lib/theme.js";
import { t } from "./lib/i18n.js";
const profileWelcome = byId("profileWelcome");
const profileCard = byId("profileCard");
const profileBioPanel = byId("profileBioPanel");
const profileRecommendationPanel = byId("profileRecommendationPanel");
let currentUserId = "";
let profileUserId = "";
initStoredTheme();
bindThemeSync();
function getUserId() {
    return new URLSearchParams(window.location.search).get("user_id");
}
function formatTime(value) {
    return new Date(value).toLocaleString();
}
function escapeHtml(value) {
    return value
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}
async function loadCurrentUser() {
    const { response, data } = await fetchCurrentUser();
    if (!response.ok) {
        window.location.href = "/login.html";
        return;
    }
    currentUserId = data.user_id;
}
function renderProfileCard(profile) {
    const avatar = resolveAvatar(profile.username, profile.icon_url, 120);
    profileCard.innerHTML = `
    <div class="profile-hero">
      <img class="profile-hero-avatar" src="${avatar}" alt="${profile.username}" />
      <div class="profile-hero-body">
        <div class="badge">${profile.is_me ? t("profile.myProfile") : t("profile.userProfile")}</div>
        <h2>${profile.username}</h2>
        <div class="profile-meta-line">${t("profile.userId", { id: profile.user_id })}</div>
        <div class="profile-meta-line">${t("profile.joinedAt", { time: formatTime(profile.created_at) })}</div>
      </div>
    </div>
  `;
}
function renderBioPanel(profile) {
    if (profile.is_me) {
        profileBioPanel.innerHTML = `
      <div class="badge">${t("profile.bio")}</div>
      <form id="profileBioForm" class="task-result-form">
        <label class="form-label" for="profileBioInput">${t("profile.personalBio")}</label>
        <textarea id="profileBioInput" class="input textarea" rows="5" maxlength="500" placeholder="${t("profile.bioPlaceholder")}">${escapeHtml(profile.bio || "")}</textarea>
        <label class="form-label" for="profileIconInput">${t("profile.avatar")}</label>
        <input id="profileIconInput" class="input" type="file" accept="image/*" />
        <div id="profileBioStatus" class="status-text"></div>
        <div class="task-form-actions">
          <button class="btn-inline btn-secondary" type="submit">${t("profile.saveProfile")}</button>
        </div>
      </form>
    `;
        const form = byId("profileBioForm");
        const bioInput = byId("profileBioInput");
        const iconInput = byId("profileIconInput");
        const status = byId("profileBioStatus");
        form.addEventListener("submit", async (event) => {
            event.preventDefault();
            status.textContent = t("profile.saving");
            const { response, data } = await updateMyProfile(bioInput.value.trim());
            if (!response.ok) {
                status.textContent = data.error || t("profile.saveFailed");
                return;
            }
            const iconFile = iconInput.files?.[0];
            if (iconFile) {
                const formData = new FormData();
                formData.append("icon", iconFile);
                const { response: iconResponse, data: iconData } = await uploadUserIcon(formData);
                if (!iconResponse.ok) {
                    status.textContent = iconData.error || t("profile.avatarUploadFailed");
                    return;
                }
            }
            status.textContent = t("profile.profileUpdated");
            await loadProfile();
        });
        return;
    }
    profileBioPanel.innerHTML = `
    <div class="badge">${t("profile.bio")}</div>
    <div class="profile-bio-copy">${profile.bio ? escapeHtml(profile.bio) : t("profile.noBio")}</div>
  `;
}
function renderRecommendationPanel(profile) {
    const recommendations = profile.recommendations || [];
    const formHtml = profile.can_recommend
        ? `
      <form id="recommendationForm" class="task-result-form">
        <label class="form-label" for="recommendationInput">${t("profile.writeRecommendation")}</label>
        <textarea id="recommendationInput" class="input textarea" rows="4" maxlength="1000" placeholder="${t("profile.recommendationPlaceholder")}"></textarea>
        <div id="recommendationStatus" class="status-text"></div>
        <div class="task-form-actions">
          <button class="btn-inline btn-secondary" type="submit">${t("profile.submitRecommendation")}</button>
        </div>
      </form>
    `
        : "";
    const listHtml = recommendations.length
        ? recommendations
            .map((item) => {
            const avatar = resolveAvatar(item.author_username, item.author_user_icon, 40);
            return `
            <div class="profile-recommendation-item">
              <div class="task-applicant-head">
                <a href="/profile.html?user_id=${encodeURIComponent(item.author_user_id)}">
                  <img class="avatar-xs" src="${avatar}" alt="${item.author_username}" />
                </a>
                <div>
                  <a class="post-author-name" href="/profile.html?user_id=${encodeURIComponent(item.author_user_id)}">${item.author_username}</a>
                  <div class="reply-meta">${formatTime(item.updated_at)}</div>
                </div>
              </div>
              <div class="profile-bio-copy">${escapeHtml(item.content)}</div>
            </div>
          `;
        })
            .join("")
        : `<div class='reply-empty'>${t("profile.noRecommendations")}</div>`;
    profileRecommendationPanel.innerHTML = `
    <div class="badge">${t("profile.recommendation")}</div>
    ${formHtml}
    <div class="task-result-list">${listHtml}</div>
  `;
    if (!profile.can_recommend) {
        return;
    }
    const form = byId("recommendationForm");
    const input = byId("recommendationInput");
    const status = byId("recommendationStatus");
    form.addEventListener("submit", async (event) => {
        event.preventDefault();
        const content = input.value.trim();
        if (!content) {
            status.textContent = t("profile.recommendationRequired");
            return;
        }
        status.textContent = t("profile.submitting");
        const { response, data } = await upsertRecommendation(profile.user_id, content);
        if (!response.ok) {
            status.textContent = data.error || t("profile.submitFailed");
            return;
        }
        status.textContent = t("profile.recommendationSaved");
        input.value = "";
        await loadProfile();
    });
}
async function loadProfile() {
    const userId = getUserId() || currentUserId;
    profileUserId = userId;
    const { response, data } = await fetchUserProfile(userId);
    if (!response.ok || !data.profile) {
        profileWelcome.textContent = data.error || t("profile.loadFailed");
        return;
    }
    const profile = data.profile;
    profileWelcome.textContent = profile.is_me
        ? t("profile.completeProfile")
        : t("profile.viewingProfile", { username: profile.username });
    renderProfileCard(profile);
    renderBioPanel(profile);
    renderRecommendationPanel(profile);
}
async function init() {
    await hydrateSiteBrand();
    await loadCurrentUser();
    await loadProfile();
}
void init();
