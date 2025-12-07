// js/auth.js

// ==== GLOBAL STATE (урезанный под auth) ====
const state = {
  accessToken: null,
  refreshToken: null,
  currentUser: null,
  serverOnline: true,
};

// ==== STORAGE ====

function saveTokens(a, r) {
  state.accessToken = a || null;
  state.refreshToken = r || null;

  if (a) localStorage.setItem("accessToken", a);
  else localStorage.removeItem("accessToken");

  if (r) localStorage.setItem("refreshToken", r);
  else localStorage.removeItem("refreshToken");
}

function loadTokens() {
  const a = localStorage.getItem("accessToken");
  const r = localStorage.getItem("refreshToken");
  if (a && r) {
    state.accessToken = a;
    state.refreshToken = r;
  }
}

function authHeaders() {
  if (!state.accessToken) return { "Content-Type": "application/json" };
  return {
    "Content-Type": "application/json",
    "Authorization": "Bearer " + state.accessToken,
  };
}

// ==== API WRAPPER ====

async function api(path, options = {}) {
  try {
    const res = await fetch("/api/v1" + path, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...(options.headers || {}),
      },
    });

    const txt = await res.text();
    let json = null;

    try {
      json = txt ? JSON.parse(txt) : null;
    } catch {
      json = null;
    }

    if (!res.ok) {
      const msg = json?.error || txt || `Ошибка ${res.status}`;
      throw new Error(msg);
    }

    // если запрос прошёл — считаем, что сервер жив
    state.serverOnline = true;
    updateServerStatus();

    return json;
  } catch (err) {
    if (err.message.includes("Failed to fetch")) {
      state.serverOnline = false;
      updateServerStatus();
      setMsg("login-msg", "❌ Сервер недоступен", "error");
    }
    throw err;
  }
}

// ==== UI HELPERS ====

function setMsg(id, msg, type = "info") {
  const el = document.getElementById(id);
  if (!el) return;
  el.textContent = msg;
  el.className = "form-msg " + type;

  if (type === "error" || type === "success") {
    setTimeout(() => {
      if (el.textContent === msg) {
        el.textContent = "";
        el.className = "form-msg";
      }
    }, 5000);
  }
}

function updateServerStatus() {
  const indicator = document.getElementById("server-status-indicator");
  const text = document.getElementById("server-status-text");
  if (!indicator || !text) return;

  if (state.serverOnline) {
    indicator.classList.remove("offline");
    indicator.classList.add("online");
    text.textContent = "Online";
    text.style.color = "#22c55e";
  } else {
    indicator.classList.remove("online");
    indicator.classList.add("offline");
    text.textContent = "Offline";
    text.style.color = "#f97373";
  }
}

// ==== AUTH HANDLERS ====

// /api/v1/register  { nickname, password } → data.access_token, data.refresh_token
async function handleRegister(e) {
  e.preventDefault();
  setMsg("register-msg", "🔄 Регистрация танкиста...", "info");

  const fd = new FormData(e.target);
  const body = {
    nickname: fd.get("nickname").trim(),
    password: fd.get("password"),
  };

  if (!body.nickname || !body.password) {
    setMsg("register-msg", "❌ Заполни ник и пароль", "error");
    return;
  }

  try {
    const res = await api("/register", {
      method: "POST",
      body: JSON.stringify(body),
    });

    const t = res.data;
    // ожидаем формат: data.access_token / data.refresh_token
    saveTokens(t.access_token, t.refresh_token);

    setMsg("register-msg", "✅ Танкист создан, вход...", "success");

    // можно дернуть /me, но не обязательно для редиректа
    try {
      await fetchMe();
    } catch {
      /* похер, всё равно идём дальше */
    }

    // после успешной регестрации → в лобби
    setTimeout(() => {
      window.location.href = "lobby.html";
    }, 600);
  } catch (err) {
    setMsg("register-msg", `❌ Ошибка: ${err.message}`, "error");
  }
}

// /api/v1/login  { nickname, password } → data.access_token, data.refresh_token
async function handleLogin(e) {
  e.preventDefault();
  setMsg("login-msg", "🔄 Вход в боевую систему...", "info");

  const fd = new FormData(e.target);
  const body = {
    nickname: fd.get("nickname").trim(),
    password: fd.get("password"),
  };

  if (!body.nickname || !body.password) {
    setMsg("login-msg", "❌ Введи ник и пароль", "error");
    return;
  }

  try {
    const res = await api("/login", {
      method: "POST",
      body: JSON.stringify(body),
    });

    // ожидаем: res.data.access_token / res.data.refresh_token
    saveTokens(res.data.access_token, res.data.refresh_token);

    setMsg("login-msg", "✅ Доступ разрешён, переходим в лобби...", "success");

    try {
      await fetchMe();
    } catch {
      /* если /me упал — не блокируем логин */
    }

    setTimeout(() => {
      window.location.href = "lobby.html";
    }, 500);
  } catch (err) {
    setMsg("login-msg", `❌ ${err.message}`, "error");
  }
}

// /api/v1/me  → user в data
async function fetchMe() {
  if (!state.accessToken) return;

  try {
    const data = await api("/me", {
      headers: authHeaders(),
    });
    state.currentUser = data.data;
  } catch (err) {
    state.currentUser = null;
  }
}

// ==== INIT ====

document.addEventListener("DOMContentLoaded", async () => {
  // Табы вход/регистрация
  const tabLogin = document.getElementById("tab-login");
  const tabRegister = document.getElementById("tab-register");
  const loginForm = document.getElementById("login-form");
  const registerForm = document.getElementById("register-form");

  tabLogin.addEventListener("click", () => {
    tabLogin.classList.add("active");
    tabRegister.classList.remove("active");
    loginForm.classList.add("active");
    registerForm.classList.remove("active");
  });

  tabRegister.addEventListener("click", () => {
    tabRegister.classList.add("active");
    tabLogin.classList.remove("active");
    registerForm.classList.add("active");
    loginForm.classList.remove("active");
  });

  // Обработчики форм
  loginForm.addEventListener("submit", handleLogin);
  registerForm.addEventListener("submit", handleRegister);

  // Пробуем поднять токены из localStorage
  loadTokens();
  updateServerStatus();

  // Если токены есть — проверяем /me и при успехе сразу в лобби
  if (state.accessToken && state.refreshToken) {
    try {
      await fetchMe();
      if (state.currentUser) {
        window.location.href = "lobby.html";
      }
    } catch {
      // если не удалось — просто сидим на auth и даём перелогиниться
      saveTokens(null, null);
    }
  }
});
