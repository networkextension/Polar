import { request, requestJson } from "./http.js";
import type { UserProfile } from "../types/session.js";

export async function fetchCurrentUser() {
  return requestJson<UserProfile>("/api/me");
}

export async function logout() {
  return request("/api/logout", { method: "POST" });
}
