// js/game.js

// ====== STATE ======
const state = {
  accessToken: null,
  refreshToken: null,
  currentUser: null,

  roomId: null,
  roomInfo: null,
  players: {},
  bullets: [],    
  meId: null,

  ws: null,
  wsConnected: false,

  keys: {
    up: false,
    down: false,
    left: false,
    right: false,
    shoot: false, 
  },

  hp: 1,
  speed: 0,
};


let ctx;
let canvas;
let lastFrameTime = 0;

// ====== UTILS ======

function loadTokens() {
  const a = localStorage.getItem("accessToken");
  const r = localStorage.getItem("refreshToken");
  if (a && r) {
    state.accessToken = a;
    state.refreshToken = r;
  }
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

async function api(path, options = {}) {
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
    throw new Error(json?.error || txt || `Ошибка ${res.status}`);
  }
  return json;
}

function setMsg(id, msg, type = "info") {
  const el = document.getElementById(id);
  if (!el) return;
  el.textContent = msg;
  el.className = "panel-msg " + type;
}

// ====== INIT / AUTH ======

async function fetchMeOrKick() {
  if (!state.accessToken || !state.refreshToken) {
    window.location.href = "auth.html";
    return;
  }

  try {
    const data = await api("/me", { headers: authHeaders() });
    state.currentUser = data.data;

    const userLabel = document.getElementById("game-username");
    if (userLabel) {
      userLabel.innerHTML =
        `<i class="fas fa-user"></i> ${state.currentUser.nickname}`;
    }
  } catch (err) {
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
    window.location.href = "auth.html";
  }
}

function getRoomIdFromUrl() {
  const url = new URL(window.location.href);
  return url.searchParams.get("room_id");
}

// async function loadRoomInfo() {
//   if (!state.roomId) return;
//   const res = await api(`/rooms/${state.roomId}`, {
//     headers: authHeaders(),
//   });
//   state.roomInfo = res.data;

//   updateRoomInfoUI();
// }
function initRoomInfoFromStorage() {
  const raw = localStorage.getItem("currentRoomInfo");
  if (!raw) return;

  try {
    state.roomInfo = JSON.parse(raw);
  } catch {
    state.roomInfo = null;
  }

  updateRoomInfoUI();
}


async function loadRoomPlayersOnce() {
  if (!state.roomId) return;
  try {
    const res = await api(`/rooms/${state.roomId}/players`, {
      headers: authHeaders(),
    });
    const players = res.data?.players || [];
    const map = {};
    players.forEach((p, i) => {
      map[p.id] = {
        id: p.id,
        nickname: p.nickname,
        x: 100 + i * 40,
        y: 100,
        angle: 0,
        color: colorForIndex(i),
      };
    });
    state.players = map;
    if (!state.meId && state.currentUser) {
      state.meId = state.currentUser.id;
    }
    updatePlayersListUI();
  } catch (_) {
    // пофиг
  }
}

function updateRoomInfoUI() {
  const info = state.roomInfo;
  if (!info) return;

  const titleEl = document.getElementById("info-room-title");
  const playersEl = document.getElementById("info-room-players");
  const timeEl = document.getElementById("info-room-time");
  const rewardEl = document.getElementById("info-room-reward");
  const topName = document.getElementById("game-room-name");

  const maxPlayers = info.max_players ?? 0;

  if (titleEl) titleEl.textContent = info.name;
  if (topName) topName.textContent = info.name;
  if (playersEl) {
    // пока число игроков берём из players
    const count = Object.keys(state.players).length;
    playersEl.textContent = `${count}/${maxPlayers}`;
  }
  if (timeEl) timeEl.textContent = `${info.duration_minutes ?? 0} мин`;
  if (rewardEl) {
    const gold = (info.gold_per_kill ?? 0) * (info.fund_modifier ?? 1);
    rewardEl.textContent = `${gold} gold`;
  }
}

// ====== PLAYERS LIST UI ======

function updatePlayersListUI() {
  const list = document.getElementById("game-players-list");
  if (!list) return;

  list.innerHTML = "";

  const playersArr = Object.values(state.players);
  if (playersArr.length === 0) {
    list.innerHTML = `
      <li class="room-player-empty">
        Игроки ещё не загрузились
      </li>
    `;
    return;
  }

  playersArr.forEach((p, idx) => {
    const li = document.createElement("li");
    const isMe = state.meId && p.id === state.meId;
    li.className = "room-player-item" + (isMe ? " you" : "");
    li.innerHTML = `
      <span class="idx">${idx + 1}.</span>
      <span class="nick">${p.nickname}</span>
      ${isMe ? '<span class="badge-you">ВЫ</span>' : ""}
    `;
    list.appendChild(li);
  });

  updateRoomInfoUI();
}

// ====== WEBSOCKET ======

function buildGameWsUrl() {
  const proto = window.location.protocol === "https:" ? "wss" : "ws";
  const host = window.location.host;

  // ⚠️ тут подгони под свой роут на сервере
  // я предполагаю: /ws/game?room_id=...&token=...
  const params = new URLSearchParams();
  params.set("room_id", state.roomId);
  if (state.accessToken) params.set("token", state.accessToken);

  return `${proto}://${host}/ws/game?${params.toString()}`;
}

function connectWs() {
  if (!state.roomId) return;

  const url = buildGameWsUrl();
  const ws = new WebSocket(url);
  state.ws = ws;

  ws.onopen = () => {
    state.wsConnected = true;
    setMsg("game-msg", "✅ Соединение с боевым сервером установлено", "success");

    // приветственный пакет (опционально)
    const hello = {
      type: "hello",
      room_id: state.roomId,
    };
    ws.send(JSON.stringify(hello));
  };

  ws.onclose = () => {
    state.wsConnected = false;
    setMsg("game-msg", "⚠️ Соединение потеряно, пробую переподключиться...", "error");
    // простейший автоподключатель
    setTimeout(() => {
      connectWs();
    }, 2000);
  };

  ws.onerror = () => {
    setMsg("game-msg", "❌ Ошибка WebSocket", "error");
  };

  ws.onmessage = (ev) => {
    let msg;
    try {
      msg = JSON.parse(ev.data);
    } catch {
      return;
    }
    handleWsMessage(msg);
  };
}

function handleWsMessage(msg) {
  if (!msg || typeof msg !== "object") return;

  if (msg.type === "state") {
    const snap = msg.data;
    if (!snap) return;

    // --- игроки ---
    if (Array.isArray(snap.players)) {
      const newMap = {};

      snap.players.forEach((p, idx) => {
        const prev = state.players[p.id];

        let angle = prev?.angle ?? -90;

        // если координаты изменились — пересчитаем направление
        if (prev && (p.x !== prev.x || p.y !== prev.y)) {
          const dx = p.x - prev.x;
          const dy = p.y - prev.y;

          if (dx !== 0 || dy !== 0) {
            angle = Math.atan2(dy, dx) * 180 / Math.PI; // в градусы
          }
        }

        newMap[p.id] = {
          id: p.id,
          nickname: prev?.nickname || p.id.slice(0, 8),
          x: p.x,
          y: p.y,
          angle,                                // <--- ВАЖНО
          color: prev?.color || colorForIndex(idx),
        };
      });

      state.players = newMap;

      if (state.currentUser) {
        state.meId = state.currentUser.id;
      }

      updatePlayersListUI();
      updateRoomInfoUI();
    }

    // --- пули ---
    if (Array.isArray(snap.bullets)) {
      state.bullets = snap.bullets.map(b => ({
        id: b.id,
        x: b.x,
        y: b.y,
      }));
    } else {
      state.bullets = [];
    }

    return;
  }

  if (msg.type === "pong") {
    // сюда потом можно пинг вернуть
    return;
  }
}




let inputSeq = 0;

function sendInputToServer() {
  if (!state.wsConnected || !state.ws) return;

  inputSeq++;

  const payload = {
    type: "input",
    input: {
      seq: inputSeq,
      up: state.keys.up,
      down: state.keys.down,
      left: state.keys.left,
      right: state.keys.right,
      shoot: state.keys.shoot,
    },
  };

  try {
    state.ws.send(JSON.stringify(payload));
  } catch (_) {}
}





// ====== RENDERING / GAME LOOP ======

function colorForIndex(i) {
  const palette = ["#22c55e", "#3b82f6", "#f97316", "#eab308", "#ec4899", "#a855f7"];
  return palette[i % palette.length];
}

function drawGrid() {
  const w = canvas.width;
  const h = canvas.height;

  ctx.fillStyle = "#020617";
  ctx.fillRect(0, 0, w, h);

  ctx.strokeStyle = "#111827";
  ctx.lineWidth = 1;

  const step = 40;
  for (let x = 0; x <= w; x += step) {
    ctx.beginPath();
    ctx.moveTo(x, 0);
    ctx.lineTo(x, h);
    ctx.stroke();
  }
  for (let y = 0; y <= h; y += step) {
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(w, y);
    ctx.stroke();
  }
}

function drawTank(tank) {
  if (!canvas) return;

  const WORLD_MAX = 20;

  const worldX = tank.x;
  const worldY = tank.y;

  const px = (worldX / WORLD_MAX) * canvas.width;
  const py = (worldY / WORLD_MAX) * canvas.height;

  // ВАЖНО:
  // bodyLength – вдоль ствола (ось X в локе)
  // bodyWidth  – поперёк, между гуслями (ось Y)
  const bodyLength = 30;
  const bodyWidth = 18;

  const treadWidth = 6;          // толщина гусли по Y
  const treadGap = 2;            // отступ от корпуса

  const barrelLength = 20;
  const barrelWidth = 4;

  const angleRad = (tank.angle || 0) * Math.PI / 180;

  ctx.save();
  ctx.translate(px, py);
  ctx.rotate(angleRad);

  // --- ГУСЛИ ---

  ctx.fillStyle = "#1f2937"; // тёмно-серый

  // верхняя гусля (по направлению вперёд/назад, над корпусом)
  ctx.fillRect(
    -bodyLength / 2 - 2,                  // x
    -bodyWidth / 2 - treadWidth - treadGap, // y
    bodyLength + 4,                       // ширина вдоль ствола
    treadWidth                            // высота (толщина) гусли
  );

  // нижняя гусля
  ctx.fillRect(
    -bodyLength / 2 - 2,
    bodyWidth / 2 + treadGap,
    bodyLength + 4,
    treadWidth
  );

  // небольшие полоски на гуслях
  ctx.strokeStyle = "rgba(15,23,42,0.9)";
  ctx.lineWidth = 1;
  const treadLines = 5;
  for (let i = 0; i < treadLines; i++) {
    const lx = -bodyLength / 2 - 2 + ((i + 1) * (bodyLength + 4)) / (treadLines + 1);

    // верхняя
    ctx.beginPath();
    ctx.moveTo(lx, -bodyWidth / 2 - treadGap);
    ctx.lineTo(lx, -bodyWidth / 2 - treadWidth - treadGap);
    ctx.stroke();

    // нижняя
    ctx.beginPath();
    ctx.moveTo(lx, bodyWidth / 2 + treadGap);
    ctx.lineTo(lx, bodyWidth / 2 + treadWidth + treadGap);
    ctx.stroke();
  }

  // --- КОРПУС ---

  ctx.fillStyle = tank.color || "#3b82f6"; // ярко-синий
  ctx.fillRect(
    -bodyLength / 2,
    -bodyWidth / 2,
    bodyLength,
    bodyWidth
  );

  ctx.strokeStyle = "rgba(15,23,42,0.9)";
  ctx.lineWidth = 1.5;
  ctx.strokeRect(
    -bodyLength / 2,
    -bodyWidth / 2,
    bodyLength,
    bodyWidth
  );

  // --- БАШНЯ ---

  ctx.beginPath();
  ctx.arc(0, 0, bodyWidth / 3, 0, Math.PI * 2);
  ctx.fillStyle = "#22c55e"; // зелёная башня
  ctx.fill();

  // --- СТВОЛ (строго вперёд по X) ---

  ctx.fillStyle = "#e5e7eb";
  ctx.fillRect(
    bodyLength / 2,          // начинаем от правого края корпуса
    -barrelWidth / 2,
    barrelLength,
    barrelWidth
  );

  ctx.restore();
}




function drawBullet(bullet) {
  if (!canvas) return;

  const WORLD_MAX = 20;

  const px = (bullet.x / WORLD_MAX) * canvas.width;
  const py = (bullet.y / WORLD_MAX) * canvas.height;

  const size = 6;

  ctx.save();
  ctx.translate(px, py);
  ctx.fillStyle = "#facc15"; // жёлтая пулька
  ctx.fillRect(-size / 2, -size / 2, size, size);
  ctx.restore();
}



function gameLoop(ts) {
  lastFrameTime = ts;
  drawGrid();
  // танки
  Object.values(state.players).forEach(drawTank);
  // пули
  state.bullets.forEach(drawBullet);
  requestAnimationFrame(gameLoop);
}




// ====== INPUT ======

function handleKeyDown(e) {
  let changed = false;

  switch (e.code) {
    case "KeyW":
    case "ArrowUp":
      if (!state.keys.up) { state.keys.up = true; changed = true; }
      break;
    case "KeyS":
    case "ArrowDown":
      if (!state.keys.down) { state.keys.down = true; changed = true; }
      break;
    case "KeyA":
    case "ArrowLeft":
      if (!state.keys.left) { state.keys.left = true; changed = true; }
      break;
    case "KeyD":
    case "ArrowRight":
      if (!state.keys.right) { state.keys.right = true; changed = true; }
      break;
    case "Space":
      e.preventDefault();
      if (!state.keys.shoot) { state.keys.shoot = true; changed = true; }
      break;
    case "Escape":
      exitToLobby();
      break;
    default:
      return;
  }

  if (changed) {
    sendInputToServer();
  }
}


function handleKeyUp(e) {
  let changed = false;

  switch (e.code) {
    case "KeyW":
    case "ArrowUp":
      if (state.keys.up) { state.keys.up = false; changed = true; }
      break;
    case "KeyS":
    case "ArrowDown":
      if (state.keys.down) { state.keys.down = false; changed = true; }
      break;
    case "KeyA":
    case "ArrowLeft":
      if (state.keys.left) { state.keys.left = false; changed = true; }
      break;
    case "KeyD":
    case "ArrowRight":
      if (state.keys.right) { state.keys.right = false; changed = true; }
      break;
    case "Space":
      if (state.keys.shoot) { state.keys.shoot = false; changed = true; }
      break;
    default:
      return;
  }

  if (changed) {
    sendInputToServer();
  }
}





function sendShoot() {
  if (!state.wsConnected || !state.ws) return;
  const msg = { type: "shoot" };
  try {
    state.ws.send(JSON.stringify(msg));
  } catch {}
}

// ====== EXIT / LEAVE ROOM ======

async function leaveRoom() {
  if (!state.roomId) return;
  try {
    await api(`/rooms/${state.roomId}/leave`, {
      method: "POST",
      headers: authHeaders(),
      body: JSON.stringify({ reason: "leave_from_game" }),
    });
  } catch (_) {
    // игнор
  }
}

async function exitToLobby() {
  await leaveRoom();
  window.location.href = "lobby.html";
}

// ====== INIT ======

document.addEventListener("DOMContentLoaded", async () => {
  canvas = document.getElementById("game-canvas");
  if (!canvas) return;
  ctx = canvas.getContext("2d");

  loadTokens();
  state.roomId = getRoomIdFromUrl();

  if (!state.roomId) {
    // если в игру зашли без id комнаты — обратно
    window.location.href = "lobby.html";
    return;
  }

  await fetchMeOrKick();
    initRoomInfoFromStorage();   // вместо loadRoomInfo()
    await loadRoomPlayersOnce(); // сюда всё ещё ходим: /rooms/{id}/players
    updateRoomInfoUI();


  document
    .getElementById("btn-exit-to-lobby")
    ?.addEventListener("click", exitToLobby);

  document
    .getElementById("btn-leave-room")
    ?.addEventListener("click", exitToLobby);

  window.addEventListener("keydown", handleKeyDown);
  window.addEventListener("keyup", handleKeyUp);

  connectWs();
  requestAnimationFrame(gameLoop);

  // пинги раз в 5 сек — если сервер это поддерживает
  setInterval(() => {
    if (!state.wsConnected || !state.ws) return;
    state.lastPingTs = performance.now();
    try {
      state.ws.send(JSON.stringify({ type: "ping" }));
    } catch {}
  }, 5000);
});

// выйти из комнаты при закрытии вкладки
window.addEventListener("beforeunload", () => {
  if (!state.roomId || !state.accessToken) return;
  try {
    navigator.sendBeacon(
      `/api/v1/rooms/${state.roomId}/leave`,
      JSON.stringify({ reason: "tab_closed" })
    );
  } catch (_) {}
});
