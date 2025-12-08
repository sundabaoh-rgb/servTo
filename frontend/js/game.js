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
    turretLeft: false,   // Z
    turretRight: false,  // X
  },

  hp: 1,
  speed: 0,

  playerOrder: [],   // фиксированный порядок слотов игроков
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

    state.playerOrder = players.map(p => p.id);

    if (!state.meId && state.currentUser) {
      state.meId = state.currentUser.id;
    }
    updatePlayersListUI();
  } catch (_) {
    // пофиг
  }
}

function updateHudHpFromState() {
  const fill = document.querySelector(".hud-bar-fill");
  if (!fill) return;

  const me = state.meId ? state.players[state.meId] : null;

  const maxHp = 100;
  let hp = maxHp;

  if (me && typeof me.hp === "number") {
    hp = me.hp;
  }

  hp = Math.max(0, Math.min(maxHp, hp));
  const ratio = hp / maxHp;

  fill.style.width = (ratio * 100) + "%";

  // цвет как у полоски над танком
  let bg;
  if (ratio >= 0.75) {
    bg = "linear-gradient(to right, #22c55e, #a3e635)";
  } else if (ratio >= 0.5) {
    bg = "linear-gradient(to right, #eab308, #fde047)";
  } else if (ratio >= 0.25) {
    bg = "linear-gradient(to right, #f97316, #fdba74)";
  } else {
    bg = "linear-gradient(to right, #ef4444, #fca5a5)";
  }
  fill.style.background = bg;
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

  // сортировка по скилу: kills desc, deaths asc, ник
  playersArr.sort((a, b) => {
    if (b.kills !== a.kills) return b.kills - a.kills;
    if (a.deaths !== b.deaths) return a.deaths - b.deaths;
    return a.nickname.localeCompare(b.nickname);
  });

  // заголовок-строка
  const header = document.createElement("li");
  header.className = "room-player-header";
  header.innerHTML = `
    <span class="idx">#</span>
    <span class="nick">Ник</span>
    <span class="stat kills">Kill</span>
    <span class="stat deaths">Dead</span>
    <span class="stat kd">K/D</span>
  `;
  list.appendChild(header);

  playersArr.forEach((p, idx) => {
    const li = document.createElement("li");
    const isMe = state.meId && p.id === state.meId;
    const kd = p.deaths > 0 ? (p.kills / p.deaths) : p.kills;

    li.className = "room-player-item" + (isMe ? " you" : "");
    li.innerHTML = `
      <span class="idx">${idx + 1}.</span>
      <span class="nick">${p.nickname}</span>
      <span class="stat kills">${p.kills}</span>
      <span class="stat deaths">${p.deaths}</span>
      <span class="stat kd">${kd.toFixed(2)}</span>
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
    setMsg("game-msg", "Соединение с сервером установлено", "success");

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



// === RELOAD HUD ===
const reloadBarEl   = document.getElementById('reload-bar');
const reloadFillEl  = reloadBarEl?.querySelector('.reload-bar-fill');
const reloadLabelEl = reloadBarEl?.querySelector('.reload-bar-label');

function updateReloadFromServer(cd, maxCd) {
  if (!reloadBarEl || !reloadFillEl || !reloadLabelEl) return;

  // защита от мусора
  if (!maxCd || maxCd <= 0) {
    reloadFillEl.style.transform = 'scaleX(1)';
    reloadBarEl.classList.remove('reloading');
    reloadLabelEl.textContent = 'Готов';
    return;
  }

  // cd — оставшееся время, 0..maxCd
  const clampedCd = Math.max(0, Math.min(maxCd, cd));
  const ratio = 1 - clampedCd / maxCd; // 0 — пусто, 1 — перезарядился

  reloadFillEl.style.transform = `scaleX(${ratio})`;

  if (clampedCd <= 0.001) {
    reloadBarEl.classList.remove('reloading');
    reloadLabelEl.textContent = 'Готов';
  } else {
    reloadBarEl.classList.add('reloading');
    reloadLabelEl.textContent = clampedCd.toFixed(1) + ' c';
  }
}




function handleWsMessage(msg) {
  if (!msg || typeof msg !== "object") return;

  if (msg.type === "state") {
  const snap = msg.data;
  if (!snap) return;

  if (Array.isArray(snap.players)) {
    const prevPlayers = state.players;
    const newMap = {};

    snap.players.forEach((p, idx) => {
  const prev = prevPlayers[p.id];

  // БЕРЁМ УГОЛ ИЗ СЕРВЕРА, а не из dx/dy
  let angle = (typeof p.angle === "number")
    ? p.angle
    : (prev?.angle ?? -90);

  const hp     = typeof p.hp === "number"    ? p.hp    : (prev?.hp    ?? 100);
  const kills  = typeof p.kills === "number" ? p.kills : (prev?.kills ?? 0);
  const deaths = typeof p.deaths === "number"? p.deaths: (prev?.deaths?? 0);
  const dead   = !!p.dead;

  const turretAngle = (typeof p.turret_angle === "number")
    ? p.turret_angle
    : (prev?.turretAngle ?? angle);

  const wasAlive = prev ? (!prev.dead && prev.hp > 0) : false;
  if (wasAlive && dead && typeof spawnExplosion === "function") {
    spawnExplosion(p.x, p.y);
  }

  const shootCd    = typeof p.shoot_cooldown === "number"
    ? p.shoot_cooldown
    : (prev?.shootCd ?? 0);

  const shootCdMax = typeof p.max_shoot_cooldown === "number"
    ? p.max_shoot_cooldown
    : (prev?.shootCdMax ?? 0.9);

  const tankObj = {
    id: p.id,
    nickname: prev?.nickname || p.id.slice(0, 8),
    x: p.x,
    y: p.y,
    angle,          // <-- уже из сервера
    turretAngle,
    color: prev?.color || colorForIndex(idx),
    hp,
    kills,
    deaths,
    dead,
    shootCd,
    shootCdMax,
  };

  newMap[p.id] = tankObj;

  if (state.currentUser && p.id === state.currentUser.id) {
    state.meId = p.id;
    updateReloadFromServer(shootCd, shootCdMax);
  }
});

    state.players = newMap;

    updatePlayersListUI();
    updateRoomInfoUI();
    updateHudHpFromState();
  }

  // пули
  if (Array.isArray(snap.bullets)) {
    state.bullets = snap.bullets.map(b => ({
      id: b.id,
      x:  b.x,
      y:  b.y,
    }));
  } else {
    state.bullets = [];
  }

  return;
}

  if (msg.type === "pong") {
    // тут потом пинг замеришь
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
      turret_left:  state.keys.turretLeft,
      turret_right: state.keys.turretRight,
      turret_to:    typeof state.turretAngle === "number" ? state.turretAngle : 999999, // если отключаем мышь то 999999
          // turret_to: 999999, // если отключаем мышь то 999999

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
  const isDead = tank.dead || tank.hp <= 0;

  const worldX = tank.x;
  const worldY = tank.y;

  const px = (worldX / WORLD_MAX) * canvas.width;
  const py = (worldY / WORLD_MAX) * canvas.height;

  const bodyLength = 30;
  const bodyWidth  = 18;

  const treadWidth = 6;
  const treadGap   = 2;

  const barrelLength = 20;
  const barrelWidth  = 4;

  // корпус
  const angleDeg = (typeof tank.angle === "number")
  ? tank.angle
  : -90;
  const angleRad  = angleDeg * Math.PI / 180;

  // башня
  const turretDeg = (typeof tank.turretAngle === "number")
    ? tank.turretAngle
    : angleDeg;
  const turretRad = turretDeg * Math.PI / 180;

  // ====== ТРУПИК ======
  if (isDead) {
    ctx.save();
    ctx.translate(px, py);
    ctx.rotate(angleRad);

    // крест из гусель
    ctx.save();
    ctx.globalAlpha = 0.9;

    ctx.rotate(Math.PI / 4);
    ctx.fillStyle = "#020617";
    ctx.fillRect(-bodyLength, -treadWidth / 2, bodyLength * 2, treadWidth);
    ctx.strokeStyle = "rgba(30,64,175,0.9)";
    ctx.lineWidth = 1;
    ctx.strokeRect(-bodyLength, -treadWidth / 2, bodyLength * 2, treadWidth);

    ctx.rotate(Math.PI / 2);
    ctx.fillStyle = "#020617";
    ctx.fillRect(-bodyLength, -treadWidth / 2, bodyLength * 2, treadWidth);
    ctx.strokeRect(-bodyLength, -treadWidth / 2, bodyLength * 2, treadWidth);

    ctx.restore();

    // обгоревший корпус
    ctx.fillStyle = "#020617";
    ctx.fillRect(-bodyLength / 2, -bodyWidth / 2, bodyLength, bodyWidth);

    ctx.strokeStyle = "rgba(148,163,184,0.6)";
    ctx.lineWidth = 1.5;
    ctx.strokeRect(-bodyLength / 2, -bodyWidth / 2, bodyLength, bodyWidth);

    // центр — вмятина
    ctx.beginPath();
    ctx.arc(0, 0, bodyWidth / 3, 0, Math.PI * 2);
    ctx.fillStyle = "rgba(15,23,42,0.95)";
    ctx.fill();

    // сломанный короткий ствол
    ctx.fillStyle = "#4b5563";
    ctx.fillRect(bodyLength / 4, -barrelWidth / 2, barrelLength / 2, barrelWidth);

    ctx.restore();
    // HP-бар для трупа не рисуем
    return;
  }

  // ====== КОРПУС ======
  ctx.save();
  ctx.translate(px, py);
  ctx.rotate(angleRad);

  // гусли
  ctx.fillStyle = "#1f2937";
  ctx.fillRect(
    -bodyLength / 2 - 2,
    -bodyWidth / 2 - treadWidth - treadGap,
    bodyLength + 4,
    treadWidth
  );
  ctx.fillRect(
    -bodyLength / 2 - 2,
    bodyWidth / 2 + treadGap,
    bodyLength + 4,
    treadWidth
  );

  ctx.strokeStyle = "rgba(15,23,42,0.9)";
  ctx.lineWidth = 1;
  const treadLines = 5;
  for (let i = 0; i < treadLines; i++) {
    const lx = -bodyLength / 2 - 2 + ((i + 1) * (bodyLength + 4)) / (treadLines + 1);

    // верхняя гусля
    ctx.beginPath();
    ctx.moveTo(lx, -bodyWidth / 2 - treadGap);
    ctx.lineTo(lx, -bodyWidth / 2 - treadWidth - treadGap);
    ctx.stroke();

    // нижняя гусля
    ctx.beginPath();
    ctx.moveTo(lx, bodyWidth / 2 + treadGap);
    ctx.lineTo(lx, bodyWidth / 2 + treadWidth + treadGap);
    ctx.stroke();
  }

  // корпус
  ctx.fillStyle = tank.color || "#15803d";
  ctx.fillRect(-bodyLength / 2, -bodyWidth / 2, bodyLength, bodyWidth);

  ctx.strokeStyle = "rgba(15,23,42,0.9)";
  ctx.lineWidth = 1.5;
  ctx.strokeRect(
    -bodyLength / 2,
    -bodyWidth / 2,
    bodyLength,
    bodyWidth
  );

  ctx.restore();

  // ====== БАШНЯ + СТВОЛ (отдельно, по turretAngle) ======
  ctx.save();
  ctx.translate(px, py);
  ctx.rotate(turretRad);

  // башня
  ctx.beginPath();
ctx.arc(0, 0, bodyWidth / 3, 0, Math.PI * 2);
// более тёмный корпус, яркая башня
ctx.fillStyle = "#16a34a";        // поярче зелёный
ctx.fill();
ctx.strokeStyle = "rgba(15,23,42,0.9)";
ctx.lineWidth = 2;
ctx.stroke();

// ствол
ctx.fillStyle = "#e5e7eb";
ctx.fillRect(
  bodyLength / 2,
  -barrelWidth / 2,
  barrelLength,
  barrelWidth
);

  ctx.restore();

  // ====== HP-БАР ======
  const maxHp = 100;
  const rawHp = typeof tank.hp === "number" ? tank.hp : maxHp;
  const hp    = Math.max(0, Math.min(maxHp, rawHp));
  const ratio = hp / maxHp;

  const barWidth  = 34;
  const barHeight = 5;
  const barX = px - barWidth / 2;
  const barY = py - 26;

  let barColor;
  if (ratio >= 0.75) {
    barColor = "rgba(34,197,94,0.9)";
  } else if (ratio >= 0.5) {
    barColor = "rgba(234,179,8,0.9)";
  } else if (ratio >= 0.25) {
    barColor = "rgba(249,115,22,0.9)";
  } else {
    barColor = "rgba(239,68,68,0.95)";
  }

  ctx.save();
  ctx.fillStyle = "rgba(15,23,42,0.85)";
  ctx.fillRect(barX, barY, barWidth, barHeight);

  ctx.fillStyle = barColor;
  ctx.fillRect(
    barX + 1,
    barY + 1,
    (barWidth - 2) * ratio,
    barHeight - 2
  );

  ctx.strokeStyle = "rgba(15,23,42,1)";
  ctx.lineWidth = 1;
  ctx.strokeRect(
    barX + 0.5,
    barY + 0.5,
    barWidth - 1,
    barHeight - 1
  );
  ctx.restore();
}






/**
 * progress: 0..1
 */
function updateReloadHUD(progress) {
  if (!reloadFillEl) return;
  const clamped = Math.max(0, Math.min(1, progress));
  reloadFillEl.style.transform = `scaleX(${clamped})`;
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
    case "KeyZ":
      if (!state.keys.turretLeft)  { state.keys.turretLeft  = true; changed = true; }
      break;
    case "KeyX":
      if (!state.keys.turretRight) { state.keys.turretRight = true; changed = true; }
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
    case "KeyZ":
      if (state.keys.turretLeft)  { state.keys.turretLeft  = false; changed = true; }
      break;
    case "KeyX":
      if (state.keys.turretRight) { state.keys.turretRight = false; changed = true; }
      break;
    default:
      return;
  }

  if (changed) {
    sendInputToServer();
  }
}

// ====== MOUSE SHOOT (LMB) ======
function handleMouseDown(e) {
  if (!canvas) return;
  if (e.button !== 0) return; // только ЛКМ

  // чтобы не выделялось/не тащилось ничего
  e.preventDefault();

  if (!state.keys.shoot) {
    state.keys.shoot = true;
    sendInputToServer();
  }
}

function handleMouseUp(e) {
  if (e.button !== 0) return;

  if (state.keys.shoot) {
    state.keys.shoot = false;
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
window.addEventListener("mousemove", e => {
  if (!canvas) return;
  if (!state.meId) return;

  const me = state.players[state.meId];
  if (!me) return;

  const rect = canvas.getBoundingClientRect();
  const mx = e.clientX - rect.left;
  const my = e.clientY - rect.top;

  const WORLD_MAX = 20;

  // координаты мыши в МИРЕ
  const worldMouseX = (mx / canvas.width)  * WORLD_MAX;
  const worldMouseY = (my / canvas.height) * WORLD_MAX;

  const dx = worldMouseX - me.x;
  const dy = worldMouseY - me.y;

  const angle = Math.atan2(dy, dx) * 180 / Math.PI;

  state.turretAngle = angle;
  sendInputToServer();
});


document.addEventListener("DOMContentLoaded", async () => {
  canvas = document.getElementById("game-canvas");
  if (!canvas) return;
  ctx = canvas.getContext("2d");

  const rect = canvas.getBoundingClientRect();
  canvas.width  = rect.width;
  canvas.height = rect.height;

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

  canvas.addEventListener("mousedown", handleMouseDown);
  window.addEventListener("mouseup", handleMouseUp);

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
