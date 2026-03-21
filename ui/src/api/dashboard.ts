import { request, requestJson } from "./http.js";
import type {
  EntryDetailResponse,
  EntryListResponse,
  ErrorResponse,
  IconUploadResponse,
  LoginHistoryResponse,
  PasskeyBeginResponse,
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
  return requestJson<ErrorResponse>("/api/passkey/register/finish", {
    method: "POST",
    headers: {
      "X-Passkey-Session": sessionId,
    },
    body: payload,
  });
}
