const THEME_STORAGE_KEY = "app-theme";
export function applyTheme(theme, persist = false) {
    const nextTheme = theme === "mono" ? "mono" : "default";
    document.documentElement.dataset.theme = nextTheme;
    if (persist) {
        localStorage.setItem(THEME_STORAGE_KEY, nextTheme);
    }
    return nextTheme;
}
export function initStoredTheme(persist = false) {
    return applyTheme(localStorage.getItem(THEME_STORAGE_KEY) || "default", persist);
}
export function bindThemeSync(onChange) {
    window.addEventListener("storage", (event) => {
        if (event.key !== THEME_STORAGE_KEY) {
            return;
        }
        const nextTheme = applyTheme(event.newValue || "default");
        onChange?.(nextTheme);
    });
}
export function getStoredThemeKey() {
    return THEME_STORAGE_KEY;
}
