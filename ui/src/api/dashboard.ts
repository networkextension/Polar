import { request, requestJson } from "./http.js";
import type {
  BotListResponse,
  BotPayload,
  EntryDetailResponse,
  EntryListResponse,
  ErrorResponse,
  IconUploadResponse,
  LLMConfigListResponse,
  LLMConfigPayload,
  LoginHistoryResponse,
  PasskeyBeginResponse,
  PasskeyListResponse,
  SiteSettings,
  SiteSettingsResponse,
  TagListResponse,
  TagPayload,
} from "../types/dashboard.js";

export async function fetchLoginHistory(limit = 5) {
  return requestJson<LoginHistoryResponse>(`/api/login-history?limit=${limit}`);
}

export async function fetchEntries(offset: number, limit = 10) {
  return requestJson<EntryListResponse>(`/api/markdown?limit=${limit}&offset=${offset}`);
}

export async function fetchEntry(id: number) {
  return requestJson<EntryDetailResponse>(`/api/markdown/${id}`);
}

export async function createTag(payload: TagPayload) {
  return requestJson<ErrorResponse>("/api/tags", {
    method: "POST",
    body: payload,
  });
}

export async function fetchTags(limit = 100, offset = 0) {
  return requestJson<TagListResponse>(`/api/tags?limit=${limit}&offset=${offset}`);
}

export async function updateTag(id: number, payload: TagPayload) {
  return requestJson<ErrorResponse>(`/api/tags/${id}`, {
    method: "PUT",
    body: payload,
  });
}

export async function removeTag(id: number) {
  return requestJson<ErrorResponse>(`/api/tags/${id}`, {
    method: "DELETE",
  });
}

export async function fetchSiteSettings() {
  return requestJson<SiteSettingsResponse>("/api/site-settings");
}

export async function fetchLLMConfigs() {
  return requestJson<LLMConfigListResponse>("/api/llm-configs");
}

export async function fetchAvailableLLMConfigs() {
  return requestJson<LLMConfigListResponse>("/api/llm-configs/available");
}

export async function createLLMConfig(payload: LLMConfigPayload) {
  return requestJson<LLMConfigListResponse>("/api/llm-configs", {
    method: "POST",
    body: payload,
  });
}

export async function testLLMConfig(payload: LLMConfigPayload) {
  return requestJson<ErrorResponse>("/api/llm-configs/test", {
    method: "POST",
    body: payload,
  });
}

export async function updateLLMConfig(
  id: number,
  payload: LLMConfigPayload & {
    update_api_key?: boolean;
  }
) {
  return requestJson<LLMConfigListResponse>(`/api/llm-configs/${id}`, {
    method: "PUT",
    body: payload,
  });
}

export async function removeLLMConfig(id: number) {
  return requestJson<ErrorResponse>(`/api/llm-configs/${id}`, {
    method: "DELETE",
  });
}

export async function fetchBotUsers() {
  return requestJson<BotListResponse>("/api/bots");
}

export async function createBotUser(payload: BotPayload) {
  return requestJson<BotListResponse>("/api/bots", {
    method: "POST",
    body: payload,
  });
}

export async function updateBotUser(id: number, payload: BotPayload) {
  return requestJson<BotListResponse>(`/api/bots/${id}`, {
    method: "PUT",
    body: payload,
  });
}

export async function removeBotUser(id: number) {
  return requestJson<ErrorResponse>(`/api/bots/${id}`, {
    method: "DELETE",
  });
}

export async function updateSiteSettings(payload: SiteSettings) {
  return requestJson<SiteSettingsResponse>("/api/site-settings", {
    method: "PUT",
    body: payload,
  });
}

export async function uploadSiteIcon(formData: FormData) {
  return requestJson<IconUploadResponse>("/api/site-settings/icon", {
    method: "POST",
    body: formData,
  });
}

export async function uploadApplePushCertificate(environment: "dev" | "prod", formData: FormData) {
  return requestJson<SiteSettingsResponse>(`/api/site-settings/apple-push-cert?env=${environment}`, {
    method: "POST",
    body: formData,
  });
}

export async function deleteApplePushCertificate(environment: "dev" | "prod") {
  return requestJson<SiteSettingsResponse>(`/api/site-settings/apple-push-cert?env=${environment}`, {
    method: "DELETE",
  });
}

export async function deleteEntry(id: number) {
  return request(`/api/markdown/${id}`, {
    method: "DELETE",
  });
}

export async function uploadUserIcon(formData: FormData) {
  return requestJson<IconUploadResponse>("/api/user/icon", {
    method: "POST",
    body: formData,
  });
}

export async function beginPasskeyRegistration() {
  return requestJson<PasskeyBeginResponse>("/api/passkey/register/begin", {
    method: "POST",
  });
}

export async function finishPasskeyRegistration(
  sessionId: string,
  payload: Record<string, unknown> | unknown[] | null
) {
  return requestJson<PasskeyListResponse>("/api/passkey/register/finish", {
    method: "POST",
    headers: {
      "X-Passkey-Session": sessionId,
    },
    body: payload,
  });
}

export async function fetchPasskeys() {
  return requestJson<PasskeyListResponse>("/api/passkeys");
}

export async function removePasskey(credentialId: string) {
  return requestJson<PasskeyListResponse>(`/api/passkeys/${encodeURIComponent(credentialId)}`, {
    method: "DELETE",
  });
}
