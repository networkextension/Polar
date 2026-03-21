const API_BASE = "";
export function buildAssetUrl(url) {
    if (!url) {
        return "";
    }
    if (url.startsWith("http://") || url.startsWith("https://")) {
        return url;
    }
    return `${API_BASE}${url}`;
}
export function makeDefaultAvatar(name, size = 64) {
    const canvas = document.createElement("canvas");
    canvas.width = size;
    canvas.height = size;
    const ctx = canvas.getContext("2d");
    if (!ctx) {
        return "";
    }
    ctx.fillStyle = "#0f172a";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.fillStyle = "#38bdf8";
    ctx.font = `bold ${Math.max(20, Math.floor(size * 0.45))}px SF Mono, Fira Code, monospace`;
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText((name || "U").trim().charAt(0).toUpperCase(), canvas.width / 2, canvas.height / 2);
    return canvas.toDataURL("image/png");
}
export function resolveAvatar(name, iconUrl, size = 64) {
    return iconUrl ? buildAssetUrl(iconUrl) : makeDefaultAvatar(name, size);
}
