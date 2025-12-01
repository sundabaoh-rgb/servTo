// ==== GLOBAL STATE ====
const state = {
  accessToken: null,
  refreshToken: null,
  currentUser: null,
  currentRoom: null,    // где ты сейчас состоишь
  selectedRoom: null,   // выделенная в таблице
roomPlayers: [],
rooms: []
};

async function loadRoomPlayers(room) {
  if (!room || !state.accessToken) {
    state.roomPlayers = [];
    renderBattleInfo();
    return;
  }

  try {
    const res = await api(`/rooms/${room.id}/players`, {
      headers: authHeaders(),
    });

    state.roomPlayers = res.data || [];
  } catch (err) {
    state.roomPlayers = [];
    logToUI(`[ERROR] load players for room ${room.name}: ${err.message}`);
  }

  renderBattleInfo();
}


function authHeaders() {
  if (!state.accessToken) return {};
  return {
    "Authorization": "Bearer " + state.accessToken
  };
}

function saveTokens(a, r) {
  state.accessToken = a;
  state.refreshToken = r;

  localStorage.setItem("accessToken", a || "");
  localStorage.setItem("refreshToken", r || "");

  renderSession();
}

function loadTokens() {
  const a = localStorage.getItem("accessToken");
  const r = localStorage.getItem("refreshToken");
  if (a && r) {
    state.accessToken = a;
    state.refreshToken = r;
  }
}

// ==== API helper ====
async function api(path, options = {}) {
  const res = await fetch("/api/v1" + path, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {})
    }
  });

  const txt = await res.text();
  let json = null;

  try {
    json = txt ? JSON.parse(txt) : null;
  } catch {}

  if (!res.ok) {
    throw new Error(json?.error || txt || "Request failed");
  }

  return json;
}

// ==== RENDER UI ====

function renderSession() {
  const shortA =
    state.accessToken
      ? state.accessToken.slice(0, 6) + "..." + state.accessToken.slice(-4)
      : "—";

  const shortR =
    state.refreshToken
      ? state.refreshToken.slice(0, 6) + "..." + state.refreshToken.slice(-4)
      : "—";

  document.getElementById("access-token").textContent = shortA;
  document.getElementById("refresh-token").textContent = shortR;

  document.getElementById("current-user").textContent =
    state.currentUser?.nickname || "—";
}

async function fetchMe() {
  if (!state.accessToken) return;

  try {
    const data = await api("/me", {
      headers: authHeaders()
    });

    state.currentUser = data.data;
  } catch {
    state.currentUser = null;
  }

  renderSession();
}

function setMsg(id, msg, isError = false) {
  const el = document.getElementById(id);
  if (!el) return;

  el.textContent = msg;
  el.classList.toggle("error", isError);
  el.classList.toggle("success", !isError);
}

// ==== AUTH ====

async function handleRegister(e) {
  e.preventDefault();
  setMsg("register-msg", "");

  const fd = new FormData(e.target);
  const body = {
    nickname: fd.get("nickname"),
    password: fd.get("password")
  };

  try {
    const res = await api("/register", {
      method: "POST",
      body: JSON.stringify(body)
    });

    const t = res.data;
    saveTokens(t.access_token, t.refresh_token);

    await fetchMe();
    await loadRooms();

    setMsg("register-msg", "Регистрация успешна", false);
  } catch (err) {
    setMsg("register-msg", err.message, true);
  }
}

async function handleLogin(e) {
  e.preventDefault();
  setMsg("login-msg", "");

  const fd = new FormData(e.target);
  const body = {
    nickname: fd.get("nickname"),
    password: fd.get("password")
  };

  try {
    const res = await api("/login", {
      method: "POST",
      body: JSON.stringify(body)
    });

    saveTokens(res.data.access_token, res.data.refresh_token);

    await fetchMe();
    await loadRooms();

    setMsg("login-msg", "Успешный вход", false);
  } catch (err) {
    setMsg("login-msg", err.message, true);
  }
}

async function handleRefresh() {
  setMsg("session-msg", "");

  if (!state.refreshToken)
    return setMsg("session-msg", "Нет refresh токена", true);

  try {
    const res = await api("/refresh", {
      method: "POST",
      body: JSON.stringify({ refresh_token: state.refreshToken })
    });

    saveTokens(res.data.access_token, res.data.refresh_token);
    await fetchMe();

    setMsg("session-msg", "Access обновлен", false);
  } catch (err) {
    setMsg("session-msg", err.message, true);
  }
}

async function handleLogout() {
  setMsg("session-msg", "");

  try {
    await fetch("/api/v1/logout", {
      method: "POST",
      headers: authHeaders()
    });
  } catch {}

  state.currentUser = null;
  state.accessToken = "";
  state.refreshToken = "";

  localStorage.removeItem("accessToken");
  localStorage.removeItem("refreshToken");

  renderSession();
  setMsg("session-msg", "Вышли из аккаунта", false);
}

// ==== ROOMS ====

async function handleCreateRoom(e) {
  e.preventDefault();
  setMsg("create-room-msg", "");

  if (!state.accessToken)
    return setMsg("create-room-msg", "Нужно войти", true);

  const fd = new FormData(e.target);

  const body = {
    name: fd.get("name"),
    max_players: Number(fd.get("max_players")),
    duration_minutes: Number(fd.get("duration_minutes")),
    gold_per_kill: Number(fd.get("gold_per_kill")),
    fund_modifier: Number(fd.get("fund_modifier"))
  };

  try {
    await api("/rooms", {
      method: "POST",
      headers: authHeaders(),
      body: JSON.stringify(body)
    });

    await loadRooms();
    setMsg("create-room-msg", "Комната создана", false);
  } catch (err) {
    setMsg("create-room-msg", err.message, true);
  }
}

async function loadRooms() {
  setMsg("rooms-msg", "");

  if (!state.accessToken)
    return setMsg("rooms-msg", "Войдите", true);

  let list;
  try {
    const res = await api("/rooms", { headers: authHeaders() });
    list = res.data || [];
  } catch (err) {
    return setMsg("rooms-msg", err.message, true);
  }

  state.rooms = list; // <--- сохраняем

  const tbody = document.querySelector("#rooms-table tbody");
  tbody.innerHTML = "";

  list.forEach(room => {
    const tr = document.createElement("tr");

    const isCurrent = state.currentRoom === room.id;

    tr.innerHTML = `
      <td>${room.id}</td>
      <td>${room.name}</td>
      <td>${room.owner_id.slice(0, 8)}...</td>
      <td>${room.max_players}</td>
      <td>${room.duration_minutes}</td>
      <td>${room.gold_per_kill}</td>
      <td>${room.fund_modifier}</td>
      <td>
        <button class="btn btn-green btn-sm join-btn"${isCurrent ? " disabled" : ""}>Join</button>
        <button class="btn btn-red btn-sm leave-btn"${isCurrent ? "" : " disabled"}>Leave</button>
      </td>
    `;

    if (isCurrent) {
      tr.classList.add("room-current"); // можно в CSS подсветить
    }

    // выбор комнаты по клику по строке
    tr.addEventListener("click", () => {
      state.selectedRoom = room;
      loadRoomPlayers(room);  // <- всегда тянем игроков выбранной комнаты
    });

    // JOIN
    tr.querySelector(".join-btn").addEventListener("click", async (ev) => {
      ev.stopPropagation();

      try {
        await api(`/rooms/${room.id}/join`, {
          method: "POST",
          headers: authHeaders()
        });

        state.currentRoom = room.id;
        state.selectedRoom = room;

        logToUI(`[JOINED] вошёл в комнату ${room.name}`);

        // обновляем комнаты и игроков после успешного входа
        await loadRooms();
        await loadRoomPlayers(room);
      } catch (err) {
        setMsg("rooms-msg", "Ошибка join: " + err.message, true);
        logToUI(`[ERROR] join ${room.name}: ${err.message}`);
      }
    });

    // LEAVE
    tr.querySelector(".leave-btn").addEventListener("click", async (ev) => {
      ev.stopPropagation();

      try {
        await api(`/rooms/${room.id}/leave`, {
          method: "POST",
          headers: authHeaders()
        });

        if (state.currentRoom === room.id) {
          state.currentRoom = null;
        }

        logToUI(`[LEFT] вышел из комнаты ${room.name}`);

        // обновляем комнаты и игроков после выхода
        await loadRooms();
        // если мы смотрим на эту же комнату — перезагрузим её игроков (их станет меньше)
        if (state.selectedRoom && state.selectedRoom.id === room.id) {
          await loadRoomPlayers(room);
        }
      } catch (err) {
        setMsg("rooms-msg", "Ошибка leave: " + err.message, true);
        logToUI(`[ERROR] leave ${room.name}: ${err.message}`);
      }
    });

    tbody.appendChild(tr);
  });

  setMsg("rooms-msg", `Комнат: ${list.length}`, false);
}


// ==== LOG PANEL ====
function logToUI(msg) {
  const box = document.getElementById("fake-chat");
  const line = document.createElement("div");
  line.textContent = `[${new Date().toLocaleTimeString()}] ${msg}`;
  box.appendChild(line);
  box.scrollTop = box.scrollHeight;
}

// ==== INIT ====
document.addEventListener("DOMContentLoaded", async () => {
  loadTokens();
  renderSession();
  renderBattleInfo();

  document.getElementById("login-form").addEventListener("submit", handleLogin);
  document.getElementById("register-form").addEventListener("submit", handleRegister);
  document.getElementById("refresh-btn").addEventListener("click", handleRefresh);
  document.getElementById("logout-btn").addEventListener("click", handleLogout);
  document.getElementById("create-room-form").addEventListener("submit", handleCreateRoom);
  document.getElementById("reload-rooms-btn").addEventListener("click", loadRooms);

  if (state.accessToken) {
    await fetchMe();
    await loadRooms();
  }

   renderBattleInfo();
  logToUI("UI загружен");
  setInterval(async () => {
    if (!state.accessToken) return;

    // текущая выбранная комната (по id)
    const selectedId = state.selectedRoom?.id || null;

    await loadRooms();

    if (selectedId) {
      const updated = state.rooms.find(r => r.id === selectedId);
      state.selectedRoom = updated || null;
      if (state.selectedRoom) {
        await loadRoomPlayers(state.selectedRoom);
      } else {
        state.roomPlayers = [];
        renderBattleInfo();
      }
    }
  }, 5000);
});

function renderBattleInfo() {
  const titleEl = document.querySelector(".battle-info-title");
  const modeEl = document.querySelector(".battle-mode");
  const timeEl = document.querySelector(".battle-time");
  const playersEl = document.querySelector(".battle-players");
  const listEl = document.querySelector(".battle-players-list");

  const room = state.selectedRoom;

  if (!room) {
    titleEl.textContent = "—";
    modeEl.textContent = "DM";
    timeEl.textContent = "15 мин";
    playersEl.textContent = "0/8";
    listEl.innerHTML = `<li class="small">Выберите комнату слева и нажмите Join.</li>`;
    return;
  }

  titleEl.textContent = room.name;
  modeEl.textContent = "DM";
  timeEl.textContent = room.duration_minutes + " мин";

  const max = room.max_players || 0;
  const count = state.roomPlayers ? state.roomPlayers.length : 0;
  playersEl.textContent = `${count}/${max}`;

  listEl.innerHTML = "";
  if (!count) {
    listEl.innerHTML = `<li class="small">В комнате пока никого.</li>`;
  } else {
    state.roomPlayers.forEach(p => {
      const li = document.createElement("li");
      li.textContent = p.nickname;
      listEl.appendChild(li);
    });
  }
}

async function loadRoomPlayers(room) {
  if (!room || !state.accessToken) {
    state.roomPlayers = [];
    renderBattleInfo();
    return;
  }

  try {
    const res = await api(`/rooms/${room.id}/players`, {
      headers: authHeaders()
    });

    state.roomPlayers = res.data || [];
  } catch (err) {
    state.roomPlayers = [];
    logToUI(`[ERROR] load players for room ${room.name}: ${err.message}`);
  }

  renderBattleInfo();
}



