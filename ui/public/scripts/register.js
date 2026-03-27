import { byId } from "./lib/dom.js";
import { buildClientHeaders } from "./lib/client.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { t } from "./lib/i18n.js";
const API_BASE = "";
const form = byId("registerForm");
const alertBox = byId("alert");
form.addEventListener("submit", async (event) => {
    event.preventDefault();
    alertBox.className = "alert";
    alertBox.textContent = "";
    const elements = form.elements;
    const payload = {
        username: elements.username.value.trim(),
        email: elements.email.value.trim(),
        password: elements.password.value,
    };
    try {
        const res = await fetch(`${API_BASE}/api/register`, {
            method: "POST",
            headers: buildClientHeaders({ "Content-Type": "application/json" }),
            credentials: "include",
            body: JSON.stringify(payload),
        });
        const data = await res.json();
        if (!res.ok) {
            alertBox.className = "alert error";
            alertBox.textContent = data.error || t("register.failed");
            return;
        }
        alertBox.className = "alert success";
        alertBox.textContent = t("register.success");
        window.setTimeout(() => {
            window.location.href = "/dashboard.html";
        }, 600);
    }
    catch {
        alertBox.className = "alert error";
        alertBox.textContent = t("common.networkError");
    }
});
async function redirectIfLoggedIn() {
    try {
        const res = await fetch("/api/me", { credentials: "include" });
        if (res.ok) {
            window.location.replace("/dashboard.html");
        }
    }
    catch {
        // Ignore bootstrap failures here.
    }
}
void redirectIfLoggedIn();
void hydrateSiteBrand();
