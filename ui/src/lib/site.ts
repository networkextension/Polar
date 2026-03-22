import { makeDefaultAvatar } from "./avatar.js";

type SiteSettings = {
  name?: string;
  description?: string;
  icon_url?: string;
};

const fallbackSite: Required<SiteSettings> = {
  name: "Polar-",
  description: "AI-assisted product prototyping workspace",
  icon_url: "",
};

function resolveSite(site?: SiteSettings): Required<SiteSettings> {
  return {
    name: site?.name?.trim() || fallbackSite.name,
    description: site?.description?.trim() || fallbackSite.description,
    icon_url: site?.icon_url || "",
  };
}

export function renderSiteBrand(site?: SiteSettings): void {
  const safeSite = resolveSite(site);
  const iconSrc = safeSite.icon_url || makeDefaultAvatar(safeSite.name, 160);

  document.querySelectorAll<HTMLElement>("[data-site-brand]").forEach((container) => {
    const nameEl = container.querySelector<HTMLElement>("[data-site-name]");
    const descEl = container.querySelector<HTMLElement>("[data-site-description]");
    const iconEl = container.querySelector<HTMLImageElement>("[data-site-icon]");

    if (nameEl) {
      nameEl.textContent = safeSite.name;
    }
    if (descEl) {
      descEl.textContent = safeSite.description;
    }
    if (iconEl) {
      iconEl.src = iconSrc;
      iconEl.alt = `${safeSite.name} 图标`;
    }
  });
}

export async function hydrateSiteBrand(): Promise<void> {
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
  } catch {
    renderSiteBrand();
  }
}
