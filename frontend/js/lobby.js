// js/lobby.js

// ===== ГЛОБАЛЬНОЕ СОСТОЯНИЕ =====
const state = {
  accessToken: null,
  refreshToken: null,
  currentUser: null,

  serverOnline: true,

  rooms: [],
  selectedRoom: null,
  currentRoomId: null,
  roomPlayers: [],

  pollingInterval: null,
};

// ===== ХРАНИЛКА ТОКЕНОВ =====

function loadTokens() {
  const a = localStorage.getItem("accessToken");
  const r = localStorage.getItem("refreshToken");
  if (a && r) {
    state.accessToken = a;
    state.refreshToken = r;
  }
}

function saveTokens(a, r) {
  state.accessToken = a || null;
  state.refreshToken = r || null;

  if (a) localStorage.setItem("accessToken", a);
  else localStorage.removeItem("accessToken");

  if (r) localStorage.setItem("refreshToken", r);
  else localStorage.removeItem("refreshToken");
}

function authHeaders() {
  if (!state.accessToken) {
    return { "Content-Type": "application/json" };
  }
  return {
    "Content-Type": "application/json",
    "Authorization": "Bearer " + state.accessToken,
  };
}

// ===== API ОБЁРТКА =====

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
    } catch {}

    if (!res.ok) {
      const msg = json?.error || txt || `Ошибка ${res.status}`;
      throw new Error(msg);
    }

    // сервер жив
    state.serverOnline = true;
    updateServerStatus();

    return json;
  } catch (err) {
    if (err.message.includes("Failed to fetch")) {
      state.serverOnline = false;
      updateServerStatus();
      setMsg("rooms-msg", "❌ Потеряно соединение с сервером", "error");
    }
    throw err;
  }
}

// ===== UI-ХЕЛПЕРЫ =====

function setMsg(id, msg, type = "info") {
  const el = document.getElementById(id);
  if (!el) return;

  el.textContent = msg;
  el.className = "panel-msg " + type;

  if (type === "error" || type === "success") {
    setTimeout(() => {
      if (el.textContent === msg) {
        el.textContent = "";
        el.className = "panel-msg";
      }
    }, 5000);
  }
}

function updateServerStatus() {
  // можно повесить что-то в topbar, если захочешь
}

// ===== AUTH / МЕ =====

async function fetchMeOrKick() {
  if (!state.accessToken || !state.refreshToken) {
    window.location.href = "auth.html";
    return;
  }

  try {
    const data = await api("/me", { headers: authHeaders() });
    state.currentUser = data.data;

    const topUser = document.getElementById("top-username");
    if (topUser) {
      topUser.innerHTML =
        `<i class="fas fa-user"></i> ${state.currentUser.nickname}`;
    }
  } catch (err) {
    // токен сдох → на экран логина
    saveTokens(null, null);
    window.location.href = "auth.html";
  }
}

async function handleLogout() {
  try {
    if (state.currentRoomId) {
      await api(`/rooms/${state.currentRoomId}/leave`, {
        method: "POST",
        headers: authHeaders(),
        body: JSON.stringify({ reason: "logout" }),
      });
    }

    await api("/logout", {
      method: "POST",
      headers: authHeaders(),
    });
  } catch (_) {
    // нам пофиг, даже если упало — выходим локально
  }

  saveTokens(null, null);
  window.location.href = "auth.html";
}

// ===== РУМЫ =====

async function loadRooms() {
  if (!state.accessToken) return;

  try {
    const res = await api("/rooms", { headers: authHeaders() });
    const baseRooms = res.data || [];

    // Подтягиваем количество игроков
    const roomsWithCounts = await Promise.all(
      baseRooms.map(async (room) => {
        try {
          const r = await api(`/rooms/${room.id}/players`, {
            headers: authHeaders(),
          });
          const players = r.data?.players || [];
          return { ...room, players_count: players.length };
        } catch {
          return room;
        }
      })
    );

    state.rooms = roomsWithCounts;
    renderRoomsTable();

    const summary = document.getElementById("rooms-summary");
    if (summary) {
      summary.textContent = `Комнат: ${state.rooms.length}`;
    }
  } catch (err) {
  }
}

function renderRoomsTable() {
  const tbody = document.getElementById("rooms-tbody");
  if (!tbody) return;

  tbody.innerHTML = "";

  if (state.rooms.length === 0) {
    const tr = document.createElement("tr");
    tr.className = "room-empty-row";
    tr.innerHTML = `
      <td colspan="5" style="text-align:center; padding:20px; color:#9ca3af;">
        <i class="fas fa-door-closed" style="margin-right:8px;"></i>
        Нет активных битв
      </td>
    `;
    tbody.appendChild(tr);
    return;
  }

  state.rooms.forEach((room) => {
    const tr = document.createElement("tr");
    tr.className = "room-row";

    const isSelected = state.selectedRoom?.id === room.id;
    const isCurrent = state.currentRoomId === room.id;
    if (isSelected) tr.classList.add("room-row-selected");
    if (isCurrent) tr.classList.add("room-row-current");

    const playersCount = room.players_count ?? room.playersCount ?? 0;
    const maxPlayers = room.max_players ?? room.maxPlayers ?? 0;

    tr.innerHTML = `
      <td>#${room.id}</td>
      <td>${room.name}</td>
      <td>${playersCount}/${maxPlayers}</td>
      <td>${room.duration_minutes ?? room.durationMinutes} мин</td>
      <td>${room.fund_modifier ?? room.fundModifier}x</td>
    `;

    tr.addEventListener("click", () => {
      selectRoom(room, tr);
    });

    tbody.appendChild(tr);
  });
}

function selectRoom(room, rowEl) {
  state.selectedRoom = room;

  document.querySelectorAll(".rooms-table tbody tr").forEach((tr) => {
    tr.classList.remove("room-row-selected");
  });

  if (rowEl) rowEl.classList.add("room-row-selected");

  loadRoomPlayers(room);
  renderRoomInfo();
}

async function loadRoomPlayers(room) {
  if (!room || !state.accessToken) {
    state.roomPlayers = [];
    renderRoomInfo();
    return;
  }

  try {
    const res = await api(`/rooms/${room.id}/players`, {
      headers: authHeaders(),
    });
    const payload = res.data || {};
    state.roomPlayers = payload.players || [];
  } catch {
    state.roomPlayers = [];
  }

  renderRoomInfo();
}

function renderRoomInfo() {
  const room = state.selectedRoom;

  const titleEl = document.getElementById("room-info-title");
  const playersEl = document.getElementById("room-info-players");
  const timeEl = document.getElementById("room-info-time");
  const rewardEl = document.getElementById("room-info-reward");
  const previewName = document.getElementById("room-preview-name");
  const previewMode = document.getElementById("room-preview-mode");
  const listEl = document.getElementById("room-players-list");
  const btn = document.getElementById("btn-join-leave");

  if (!room) {
    if (titleEl) titleEl.textContent = "—";
    if (playersEl) playersEl.textContent = "0/0";
    if (timeEl) timeEl.textContent = "—";
    if (rewardEl) rewardEl.textContent = "—";
    if (previewName) previewName.textContent = "Выберите битву";
    if (previewMode) previewMode.textContent = "Режим: DM";

    if (listEl) {
      listEl.innerHTML = `
        <li class="room-player-empty">
          <i class="fas fa-arrow-left"></i>
          Выберите битву в списке
        </li>
      `;
    }

    if (btn) {
      btn.disabled = true;
      btn.classList.remove("btn-red");
      btn.classList.add("btn-green");
      btn.innerHTML = `<i class="fas fa-play"></i><span>ВСТУПИТЬ В БИТВУ</span>`;
    }

    return;
  }

  const playersCount = state.roomPlayers.length;
  const maxPlayers = room.max_players ?? 0;
  const full = playersCount >= maxPlayers && maxPlayers > 0;
  const inRoom = state.currentRoomId === room.id;

  if (titleEl) titleEl.textContent = room.name;
  if (playersEl) playersEl.textContent = `${playersCount}/${maxPlayers}`;
  if (timeEl) timeEl.textContent = `${room.duration_minutes ?? 0} мин`;
  if (rewardEl) {
    const gold = (room.gold_per_kill ?? 0) * (room.fund_modifier ?? 1);
    rewardEl.textContent = `${gold} gold`;
  }
  if (previewName) previewName.textContent = room.name;
  if (previewMode) previewMode.textContent = "Режим: Deathmatch";

  if (listEl) {
    listEl.innerHTML = "";
    if (playersCount === 0) {
      listEl.innerHTML = `
        <li class="room-player-empty">
          <i class="fas fa-user-slash"></i>
          На этой битве пока никого
        </li>
      `;
    } else {
      state.roomPlayers.forEach((p, idx) => {
        const li = document.createElement("li");
        const isYou = state.currentUser && p.id === state.currentUser.id;
        li.className = "room-player-item" + (isYou ? " you" : "");
        li.innerHTML = `
          <span class="idx">${idx + 1}.</span>
          <span class="nick">${p.nickname}</span>
          ${isYou ? '<span class="badge-you">ВЫ</span>' : ""}
        `;
        listEl.appendChild(li);
      });
    }
  }

  if (!btn) return;

  if (inRoom) {
    btn.disabled = false;
    btn.classList.remove("btn-green");
    btn.classList.add("btn-red");
    btn.innerHTML = `<i class="fas fa-door-open"></i><span>ПОКИНУТЬ БИТВУ</span>`;
  } else if (full) {
    btn.disabled = true;
    btn.classList.remove("btn-red");
    btn.classList.add("btn-green");
    btn.innerHTML = `<i class="fas fa-lock"></i><span>БИТВА ЗАПОЛНЕНА</span>`;
  } else {
    btn.disabled = false;
    btn.classList.remove("btn-red");
    btn.classList.add("btn-green");
    btn.innerHTML = `<i class="fas fa-play"></i><span>ВСТУПИТЬ В БИТВУ</span>`;
  }
}

// ===== JOIN / LEAVE =====

async function handleJoinLeave() {
  const room = state.selectedRoom;
  if (!room) {
    setMsg("battle-msg", "❌ Сначала выбери битву", "error");
    return;
  }

  if (!state.accessToken) {
    setMsg("battle-msg", "❌ Не авторизован", "error");
    window.location.href = "auth.html";
    return;
  }

  // уже в битве → выходим
  if (state.currentRoomId === room.id) {
    try {
      setMsg("battle-msg", "🔄 Выход из битвы...", "info");
      await api(`/rooms/${room.id}/leave`, {
        method: "POST",
        headers: authHeaders(),
      });
      state.currentRoomId = null;
      await Promise.all([loadRooms(), loadRoomPlayers(room)]);
      setMsg("battle-msg", "ℹ️ Вы покинули битву", "success");
    } catch (err) {
      setMsg("battle-msg", `❌ Ошибка выхода: ${err.message}`, "error");
    }
    return;
  }

  // ещё не в этой битве → входим
  try {
    setMsg("battle-msg", "🔄 Вступление в битву...", "info");
    await api(`/rooms/${room.id}/join`, {
      method: "POST",
      headers: authHeaders(),
    });
    state.currentRoomId = room.id;
    await Promise.all([loadRooms(), loadRoomPlayers(room)]);
    setMsg("battle-msg", "✅ Вы в битве", "success");
    localStorage.setItem("currentRoomInfo", JSON.stringify(room));
    window.location.href = `game.html?room_id=${room.id}`;
  } catch (err) {
    setMsg("battle-msg", `❌ Ошибка входа: ${err.message}`, "error");
  }
}

// ===== ЧАТ (пока локальная заглушка) =====

function initFakeChat() {
  const box = document.getElementById("chat-messages");
  if (!box) return;

  const push = (text, type = "system") => {
    const time = new Date().toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
    const row = document.createElement("div");
    row.className = `chat-row ${type}`;
    row.innerHTML = `<span class="time">[${time}]</span> ${text}`;
    box.appendChild(row);
    box.scrollTop = box.scrollHeight;
  };

  // пара фейковых сообщений для атмосферы
  push("Система: Лобби загружено. Добро пожаловать.", "system");
  push("Совет: выбери битву в центре и жми «Вступить в битву».", "system");

  const form = document.getElementById("chat-form");
  const input = document.getElementById("chat-input");
  if (!form || !input) return;

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const txt = input.value.trim();
    if (!txt) return;
    const nick = state.currentUser?.nickname || "Ты";
    push(`${nick}: ${txt}`, "self");
    input.value = "";
  });
}

// ===== INIT =====

document.addEventListener("DOMContentLoaded", async () => {
  loadTokens();

  // защита лобби, если нет токенов
  if (!state.accessToken || !state.refreshToken) {
    window.location.href = "auth.html";
    return;
  }

  // верхние кнопки
  const btnGarage = document.getElementById("btn-garage");
  if (btnGarage) {
    btnGarage.addEventListener("click", () => {
      // пока заглушка
      alert("Гараж ещё не завезли. Потом прикрутим.");
    });
  }

  const btnLogout = document.getElementById("btn-logout");
  if (btnLogout) {
    btnLogout.addEventListener("click", handleLogout);
  }

  const btnRefresh = document.getElementById("btn-refresh-rooms");
  if (btnRefresh) {
    btnRefresh.addEventListener("click", loadRooms);
  }

  const btnJoinLeave = document.getElementById("btn-join-leave");
  if (btnJoinLeave) {
    btnJoinLeave.addEventListener("click", handleJoinLeave);
  }

  initFakeChat();

  await fetchMeOrKick();
  await loadRooms();
  renderRoomInfo();

  // автообновление списка каждые 5 сек
  state.pollingInterval = setInterval(async () => {
    if (!state.accessToken) return;
    try {
      await loadRooms();
      if (state.selectedRoom) {
        await loadRoomPlayers(state.selectedRoom);
      }
    } catch {
      // молча
    }
  }, 5000);
});

const infoView   = document.getElementById('battleinfo-view-info');
const createView = document.getElementById('battleinfo-view-create');

document.getElementById('btn-open-create-room').onclick = () => {
  infoView.classList.add('hidden');
  createView.classList.remove('hidden');
};

document.getElementById('btn-cancel-create-room').onclick = () => {
  createView.classList.add('hidden');
  infoView.classList.remove('hidden');
};

// ========================
//  СОЗДАНИЕ БИТВЫ (НОВОЕ ЛОББИ, СТАРЫЙ БЭКЕНД)
// ========================
(function initCreateRoomUI() {
  const infoView   = document.getElementById("battleinfo-view-info");
  const createView = document.getElementById("battleinfo-view-create");

  const btnOpenCreate   = document.getElementById("btn-open-create-room");
  const btnCancelCreate = document.getElementById("btn-cancel-create-room");
  const createForm      = document.getElementById("create-room-form");
  const battleMsgEl     = document.getElementById("battle-msg");

  if (!infoView || !createView || !btnOpenCreate || !btnCancelCreate || !createForm) {
    return;
  }

  function showBattleMsg(text, type = "info") {
    if (!battleMsgEl) return;
    battleMsgEl.textContent = text;
    battleMsgEl.className = "panel-msg";
    if (type === "success") battleMsgEl.classList.add("text-success");
    if (type === "error")   battleMsgEl.classList.add("text-danger");
    if (type === "warning") battleMsgEl.classList.add("text-warning");
  }

  function switchToCreateView() {
    infoView.classList.add("hidden");
    createView.classList.remove("hidden");
    showBattleMsg("");
  }

  function switchToInfoView() {
    createView.classList.add("hidden");
    infoView.classList.remove("hidden");
  }

  // открыть форму создания
  btnOpenCreate.addEventListener("click", (e) => {
    e.preventDefault();
    switchToCreateView();
  });

  // отмена
  btnCancelCreate.addEventListener("click", (e) => {
    e.preventDefault();
    switchToInfoView();
  });

  // сабмит формы -> /api/v1/rooms
  createForm.addEventListener("submit", async (e) => {
    e.preventDefault();

    if (!state.accessToken) {
      showBattleMsg("❌ Для создания битвы нужно войти", "error");
      return;
    }

    if (state.currentRoom) {
      showBattleMsg("❌ Сначала покиньте текущую битву", "error");
      return;
    }

    const fd = new FormData(createForm);
    const body = {
      name:            fd.get("name").trim(),
      max_players:     Number(fd.get("max_players")),
      duration_minutes:Number(fd.get("duration_minutes")),
      gold_per_kill:   Number(fd.get("gold_per_kill")),
      fund_modifier:   Number(fd.get("fund_modifier"))
    };

    if (!body.name) {
      showBattleMsg("❌ Название битвы не может быть пустым", "error");
      return;
    }

    showBattleMsg("🔄 Создание битвы...", "info");

    try {
      await api("/rooms", {
        method: "POST",
        headers: authHeaders(),
        body: JSON.stringify(body)
      });

      // обновляем список комнат
      await loadRooms();

      showBattleMsg("✅ Битва создана!", "success");

      createForm.reset();
      switchToInfoView();

      // по-хорошему ещё выбрать созданную комнату — если бэк возвращает её id,
      // можно доработать: сначала получить res = await api(...), потом res.data.id
    } catch (err) {
      showBattleMsg(`❌ Ошибка создания: ${err.message}`, "error");
    }
  });
})();
