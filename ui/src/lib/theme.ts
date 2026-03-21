const THEME_STORAGE_KEY = "app-theme";

export type ThemeName = "default" | "mono";

export function applyTheme(theme: string, persist = false): ThemeName {
  const nextTheme: ThemeName = theme === "mono" ? "mono" : "default";
  document.documentElement.dataset.theme = nextTheme;
  if (persist) {
    localStorage.setItem(THEME_STORAGE_KEY, nextTheme);
  }
  return nextTheme;
}

export function initStoredTheme(persist = false): ThemeName {
  return applyTheme(localStorage.getItem(THEME_STORAGE_KEY) || "default", persist);
}

export function bindThemeSync(onChange?: (theme: ThemeName) => void): void {
  window.addEventListener("storage", (event) => {
    if (event.key !== THEME_STORAGE_KEY) {
      return;
    }
    const nextTheme = applyTheme(event.newValue || "default");
    onChange?.(nextTheme);
  });
}

export function getStoredThemeKey(): string {
  return THEME_STORAGE_KEY;
}
