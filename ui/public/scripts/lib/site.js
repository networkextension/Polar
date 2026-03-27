import { makeDefaultAvatar } from "./avatar.js";
import { applyI18n, t } from "./i18n.js";
const fallbackSite = {
    name: "Polar-",
    description: "AI-assisted product prototyping workspace",
    icon_url: "",
};
function resolveSite(site) {
    return {
        name: site?.name?.trim() || fallbackSite.name,
        description: site?.description?.trim() || fallbackSite.description,
        icon_url: site?.icon_url || "",
    };
}
export function renderSiteBrand(site) {
    const safeSite = resolveSite(site);
    const iconSrc = safeSite.icon_url || makeDefaultAvatar(safeSite.name, 160);
    document.querySelectorAll("[data-site-brand]").forEach((container) => {
        const nameEl = container.querySelector("[data-site-name]");
        const descEl = container.querySelector("[data-site-description]");
        const iconEl = container.querySelector("[data-site-icon]");
        if (nameEl) {
            nameEl.textContent = safeSite.name;
        }
        if (descEl) {
            descEl.textContent = safeSite.description;
        }
        if (iconEl) {
            iconEl.src = iconSrc;
            iconEl.alt = `${safeSite.name} ${t("brand.icon")}`;
        }
    });
}
export async function hydrateSiteBrand() {
    applyI18n();
    if (!document.querySelector("[data-site-brand]")) {
        return;
    }
    try {
        const response = await fetch("/api/site-settings", { credentials: "include" });
        if (!response.ok) {
            renderSiteBrand();
            return;
        }
        const data = await response.json();
        renderSiteBrand(data.site);
    }
    catch {
        renderSiteBrand();
    }
}
