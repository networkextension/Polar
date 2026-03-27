export function getClientDeviceType(): string {
  return "browser";
}

export function getClientDeviceId(): string {
  try {
    const existing = window.localStorage.getItem("device_id")?.trim() || "";
    if (existing) {
      return existing;
    }
    const next = `browser-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
    window.localStorage.setItem("device_id", next);
    return next;
  } catch {
    return "default:browser";
  }
}

export function getStoredPushToken(): string {
  try {
    return window.localStorage.getItem("push_token")?.trim() || "";
  } catch {
    return "";
  }
}

export function getStoredLanguage(): string {
  try {
    const value = window.localStorage.getItem("polar_lang")?.trim() || "";
    return value === "zh-CN" ? "zh-CN" : "en";
  } catch {
    return "en";
  }
}

export function buildClientHeaders(headers: HeadersInit = {}): Headers {
  const merged = new Headers(headers);
  merged.set("X-Device-Type", getClientDeviceType());
  merged.set("X-Device-Id", getClientDeviceId());
  merged.set("X-Language", getStoredLanguage());

  const pushToken = getStoredPushToken();
  if (pushToken) {
    merged.set("X-Push-Token", pushToken);
  }

  return merged;
}

export function formatDeviceType(deviceType?: string, tFn?: (key: string) => string): string {
  switch (deviceType) {
    case "ios":
      return "iOS";
    case "android":
      return "Android";
    default:
      return tFn ? tFn("device.browser") : "Browser";
  }
}
