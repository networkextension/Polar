import { byId } from "./lib/dom.js";
import { base64URLToBuffer, credentialToJSON } from "./lib/passkey.js";
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
            headers: { "Content-Type": "application/json" },
            credentials: "include",
            body: JSON.stringify(payload),
        });
        const data = await res.json();
        if (!res.ok) {
            alertBox.className = "alert error";
            alertBox.textContent = data.error || "登录失败";
            return;
        }
        alertBox.className = "alert success";
        alertBox.textContent = "登录成功，正在跳转...";
        window.setTimeout(() => {
            window.location.href = "/dashboard.html";
        }, 600);
    }
    catch {
        alertBox.className = "alert error";
        alertBox.textContent = "网络错误，请稍后重试";
    }
});
passkeyLoginBtn.addEventListener("click", async () => {
    alertBox.className = "alert";
    alertBox.textContent = "";
    if (!window.PublicKeyCredential) {
        passkeyStatus.textContent = "当前浏览器不支持 Passkey。";
        return;
    }
    const elements = form.elements;
    const email = elements.email.value.trim();
    if (!email) {
        passkeyStatus.textContent = "请先输入邮箱地址。";
        return;
    }
    passkeyStatus.textContent = "正在启动 Passkey...";
    try {
        const beginRes = await fetch(`${API_BASE}/api/passkey/login/begin`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            credentials: "include",
            body: JSON.stringify({ email }),
        });
        const beginResult = await beginRes.json();
        if (!beginRes.ok) {
            passkeyStatus.textContent = beginResult.error || "无法发起 Passkey 登录";
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
                "Content-Type": "application/json",
                "X-Passkey-Session": beginResult.session_id,
            },
            credentials: "include",
            body: JSON.stringify(payload),
        });
        const finishResult = await finishRes.json();
        if (finishRes.ok) {
            passkeyStatus.textContent = "Passkey 登录成功，正在跳转...";
            window.setTimeout(() => {
                window.location.href = "/dashboard.html";
            }, 600);
            return;
        }
        passkeyStatus.textContent = finishResult.error || "Passkey 登录失败";
    }
    catch {
        passkeyStatus.textContent = "网络错误，请稍后重试";
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
