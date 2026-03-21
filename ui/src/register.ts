import { byId } from "./lib/dom.js";

const API_BASE = "";
const form = byId<HTMLFormElement>("registerForm");
const alertBox = byId<HTMLElement>("alert");

type RegisterFormElements = HTMLFormControlsCollection & {
  username: HTMLInputElement;
  email: HTMLInputElement;
  password: HTMLInputElement;
};

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  alertBox.className = "alert";
  alertBox.textContent = "";

  const elements = form.elements as RegisterFormElements;
  const payload = {
    username: elements.username.value.trim(),
    email: elements.email.value.trim(),
    password: elements.password.value,
  };

  try {
    const res = await fetch(`${API_BASE}/api/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(payload),
    });
    const data = await res.json();

    if (!res.ok) {
      alertBox.className = "alert error";
      alertBox.textContent = data.error || "注册失败";
      return;
    }

    alertBox.className = "alert success";
    alertBox.textContent = "注册成功，正在跳转...";
    window.setTimeout(() => {
      window.location.href = "/dashboard.html";
    }, 600);
  } catch {
    alertBox.className = "alert error";
    alertBox.textContent = "网络错误，请稍后重试";
  }
});

async function redirectIfLoggedIn(): Promise<void> {
  try {
    const res = await fetch("/api/me", { credentials: "include" });
    if (res.ok) {
      window.location.replace("/dashboard.html");
    }
  } catch {
    // Ignore bootstrap failures here.
  }
}

void redirectIfLoggedIn();
