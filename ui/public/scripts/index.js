import { byId } from "./lib/dom.js";
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
    statusText.textContent = "请选择操作";
}
void bootstrap();
