import { byId } from "./lib/dom.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { t } from "./lib/i18n.js";
const statusText = byId("statusText");
async function bootstrap() {
    try {
        const res = await fetch("/api/me", { credentials: "include" });
        if (res.ok) {
            window.location.replace("/dashboard.html");
            return;
        }
    }
    catch {
        // Ignore and show entry links.
    }
    statusText.textContent = t("index.selectAction");
}
void bootstrap();
void hydrateSiteBrand();
