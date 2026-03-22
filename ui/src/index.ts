import { byId } from "./lib/dom.js";
import { hydrateSiteBrand } from "./lib/site.js";

const statusText = byId<HTMLElement>("statusText");

async function bootstrap(): Promise<void> {
  try {
    const res = await fetch("/api/me", { credentials: "include" });
    if (res.ok) {
      window.location.replace("/dashboard.html");
      return;
    }
  } catch {
    // Ignore and show entry links.
  }

  statusText.textContent = "请选择操作";
}

void bootstrap();
void hydrateSiteBrand();
