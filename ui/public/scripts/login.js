import { byId } from "./lib/dom.js";
import { base64URLToBuffer, credentialToJSON } from "./lib/passkey.js";
import { buildClientHeaders } from "./lib/client.js";
import { hydrateSiteBrand } from "./lib/site.js";
import { t } from "./lib/i18n.js";
const API_BASE = "";
const form = byId("loginForm");
const alertBox = byId("alert");
const passkeyLoginBtn = byId("passkeyLoginBtn");
const passkeyStatus = byId("passkeyStatus");
form.addEventListener("submit", async (event) => {
    event.preventDefault();
    alertBox.className = "alert";
    alertBox.textContent = "";
    const elements = form.elements;
    const payload = {
        email: elements.email.value.trim(),
        password: elements.password.value,
    };
    try {
        const res = await fetch(`${API_BASE}/api/login`, {
            method: "POST",
            headers: buildClientHeaders({ "Content-Type": "application/json" }),
            credentials: "include",
            body: JSON.stringify(payload),
        });
        const data = await res.json();
        if (!res.ok) {
            alertBox.className = "alert error";
            alertBox.textContent = data.error || t("login.failed");
            return;
        }
        alertBox.className = "alert success";
        alertBox.textContent = t("login.success");
        window.setTimeout(() => {
            window.location.href = "/dashboard.html";
        }, 600);
    }
    catch {
        alertBox.className = "alert error";
        alertBox.textContent = t("common.networkError");
    }
});
passkeyLoginBtn.addEventListener("click", async () => {
    alertBox.className = "alert";
    alertBox.textContent = "";
    if (!window.PublicKeyCredential) {
        passkeyStatus.textContent = t("login.passkeyNotSupported");
        return;
    }
    const elements = form.elements;
    const email = elements.email.value.trim();
    if (!email) {
        passkeyStatus.textContent = t("login.passkeyEnterEmail");
        return;
    }
    passkeyStatus.textContent = t("login.passkeyStarting");
    try {
        const beginRes = await fetch(`${API_BASE}/api/passkey/login/begin`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            credentials: "include",
            body: JSON.stringify({ email }),
        });
        const beginResult = await beginRes.json();
        if (!beginRes.ok) {
            passkeyStatus.textContent = beginResult.error || t("login.passkeyBeginFailed");
            return;
        }
        const publicKey = beginResult.publicKey;
        publicKey.challenge = base64URLToBuffer(publicKey.challenge);
        if (publicKey.allowCredentials) {
            publicKey.allowCredentials = publicKey.allowCredentials.map((cred) => ({
                ...cred,
                id: base64URLToBuffer(cred.id),
            }));
        }
        const credential = await navigator.credentials.get({ publicKey });
        const payload = credentialToJSON(credential);
        const finishRes = await fetch(`${API_BASE}/api/passkey/login/finish`, {
            method: "POST",
            headers: {
                ...Object.fromEntries(buildClientHeaders({ "Content-Type": "application/json" })),
                "X-Passkey-Session": beginResult.session_id,
            },
            credentials: "include",
            body: JSON.stringify(payload),
        });
        const finishResult = await finishRes.json();
        if (finishRes.ok) {
            passkeyStatus.textContent = t("login.passkeySuccess");
            window.setTimeout(() => {
                window.location.href = "/dashboard.html";
            }, 600);
            return;
        }
        passkeyStatus.textContent = finishResult.error || t("login.passkeyFailed");
    }
    catch {
        passkeyStatus.textContent = t("common.networkError");
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
