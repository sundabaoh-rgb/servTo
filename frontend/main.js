// ==== GLOBAL STATE ====
const state = {
    accessToken: null,
    refreshToken: null,
    currentUser: null,
    currentRoom: null,
    selectedRoom: null,
    roomPlayers: [],
    rooms: [],
    serverOnline: true,
    onlineCount: 0,
    battlingCount: 0,
    pollingInterval: null,
    autoRefresh: true,
    mobileMenuOpen: false
};

function authHeaders() {
    if (!state.accessToken) return {};
    return {
        "Authorization": "Bearer " + state.accessToken,
        "Content-Type": "application/json"
    };
}

function saveTokens(a, r) {
    state.accessToken = a;
    state.refreshToken = r;
    localStorage.setItem("accessToken", a || "");
    localStorage.setItem("refreshToken", r || "");
    renderSession();
    updateUIByAuthState();
    updateUserStatus();
}

function loadTokens() {
    const a = localStorage.getItem("accessToken");
    const r = localStorage.getItem("refreshToken");
    if (a && r) {
        state.accessToken = a;
        state.refreshToken = r;
    }
}

async function api(path, options = {}) {
    try {
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
            throw new Error(json?.error || txt || `Ошибка ${res.status}`);
        }

        return json;
    } catch (err) {
        if (err.message.includes("Failed to fetch")) {
            state.serverOnline = false;
            updateServerStatus();
            logToUI("❌ Потеряно соединение с сервером", "error");
        }
        throw err;
    }
}

// ==== UI FUNCTIONS ====

function renderSession() {
    document.getElementById("access-token").textContent = 
        state.accessToken ? state.accessToken: "Не авторизован";
    document.getElementById("refresh-token").textContent = 
        state.refreshToken ? state.refreshToken : "Не авторизован";
    document.getElementById("current-user").textContent = 
        state.currentUser?.nickname || "Гость";
    
    updateUserStatus();
}

function updateUserStatus() {
    const statusEl = document.getElementById("user-status");
    if (!statusEl) return;
    
    const dot = statusEl.querySelector(".status-dot");
    const text = statusEl.querySelector(".status-text");
    
    if (state.currentUser) {
        if (state.currentRoom) {
            dot.className = "status-dot online";
            text.textContent = "В бою";
            text.style.color = "#2ecc71";
        } else {
            dot.className = "status-dot online";
            text.textContent = "Онлайн";
            text.style.color = "#2ecc71";
        }
    } else {
        dot.className = "status-dot offline";
        text.textContent = "Оффлайн";
        text.style.color = "#95a5a6";
    }
}

function updateUIByAuthState() {
    const isLoggedIn = !!state.currentUser;
    const loginBtn = document.querySelector('#login-form .btn');
    
    if (isLoggedIn) {
        loginBtn.innerHTML = '<i class="fas fa-check"></i> <span>В БОЮ КАК ' + state.currentUser.nickname + '</span>';
        loginBtn.disabled = true;
        loginBtn.classList.remove('btn-green');
        loginBtn.classList.add('btn-blue');
    } else {
        loginBtn.innerHTML = '<i class="fas fa-play"></i> <span>В БОЙ!</span>';
        loginBtn.disabled = false;
        loginBtn.classList.remove('btn-blue');
        loginBtn.classList.add('btn-green');
    }
}

function updateServerStatus() {
    const indicator = document.getElementById("server-status-indicator");
    const text = document.getElementById("server-status-text");
    
    if (state.serverOnline) {
        indicator.className = "status-indicator online";
        text.textContent = "Online";
        text.style.color = "#2ecc71";
    } else {
        indicator.className = "status-indicator";
        indicator.style.background = "#e74c3c";
        indicator.style.boxShadow = "0 0 10px #e74c3c";
        text.textContent = "Offline";
        text.style.color = "#e74c3c";
    }
}

function updateFooterStats() {
    document.getElementById("footer-online").textContent = state.onlineCount;
    document.getElementById("footer-rooms").textContent = state.rooms.length;
    document.getElementById("footer-battling").textContent = state.battlingCount;
    document.getElementById("rooms-count").textContent = state.rooms.length;
}

function setMsg(id, msg, type = "info") {
    const el = document.getElementById(id);
    if (!el) return;

    el.textContent = msg;
    el.className = "form-msg " + type;
    
    if (type !== "info") {
        setTimeout(() => {
            if (el.textContent === msg) {
                el.textContent = "";
                el.className = "form-msg";
            }
        }, 5000);
    }
}

function logToUI(msg, type = "info") {
    const box = document.getElementById("fake-chat");
    const time = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    
    const entry = document.createElement("div");
    entry.className = `log-entry ${type}`;
    entry.innerHTML = `<span class="time">[${time}]</span> ${msg}`;
    
    box.appendChild(entry);
    box.scrollTop = box.scrollHeight;
    
    if (box.children.length > 50) {
        box.removeChild(box.firstChild);
    }
}

// ==== MOBILE MENU ====
function toggleMobileMenu() {
    const menu = document.getElementById("topbar-menu");
    const btn = document.getElementById("mobile-menu-btn");
    
    state.mobileMenuOpen = !state.mobileMenuOpen;
    
    if (state.mobileMenuOpen) {
        menu.classList.add("active");
        btn.innerHTML = '<i class="fas fa-times"></i>';
        btn.style.background = "linear-gradient(to bottom, #ff5500 0%, #cc4400 100%)";
        btn.style.borderColor = "#ff9900";
        btn.style.color = "#0d1117";
    } else {
        menu.classList.remove("active");
        btn.innerHTML = '<i class="fas fa-bars"></i>';
        btn.style.background = "transparent";
        btn.style.borderColor = "#ff5500";
        btn.style.color = "#ff9900";
    }
}

// ==== AUTH ====

async function handleRegister(e) {
    e.preventDefault();
    setMsg("register-msg", "🔄 Регистрация нового танкиста...", "info");

    const fd = new FormData(e.target);
    const body = {
        nickname: fd.get("nickname").trim(),
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

        setMsg("register-msg", "✅ Танкист зарегистрирован!", "success");
        logToUI(`✅ Добро пожаловать в экипаж, ${body.nickname}!`, "success");
        e.target.reset();
        
    } catch (err) {
        setMsg("register-msg", `❌ Ошибка регистрации: ${err.message}`, "error");
        logToUI(`❌ Ошибка регистрации: ${err.message}`, "error");
    }
}

async function handleLogin(e) {
    e.preventDefault();
    setMsg("login-msg", "🔄 Вход в боевую систему...", "info");

    const fd = new FormData(e.target);
    const body = {
        nickname: fd.get("nickname").trim(),
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

        setMsg("login-msg", "✅ Доступ разрешён!", "success");
        logToUI(`✅ Добро пожаловать на поле боя, ${body.nickname}!`, "success");
        e.target.reset();
        
    } catch (err) {
        setMsg("login-msg", `❌ ${err.message}`, "error");
        logToUI(`❌ Ошибка входа: ${err.message}`, "error");
    }
}

async function handleRefresh() {
    setMsg("session-msg", "🔄 Обновление боевой сессии...", "info");

    if (!state.refreshToken) {
        setMsg("session-msg", "❌ Нет refresh токена", "error");
        return;
    }

    try {
        const res = await api("/refresh", {
            method: "POST",
            body: JSON.stringify({ refresh_token: state.refreshToken })
        });

        saveTokens(res.data.access_token, res.data.refresh_token);
        await fetchMe();

        setMsg("session-msg", "✅ Сессия обновлена", "success");
        logToUI("✅ Боевая сессия продлена", "success");
    } catch (err) {
        setMsg("session-msg", `❌ Ошибка обновления: ${err.message}`, "error");
    }
}

async function handleLogout() {
    setMsg("session-msg", "🔄 Выход из боевой системы...", "info");

    try {
        if (state.currentRoom) {
            await leaveCurrentRoom();
        }
        
        await fetch("/api/v1/logout", {
            method: "POST",
            headers: authHeaders()
        });
    } catch (err) {
        console.log("Logout error:", err);
    }

    state.currentUser = null;
    state.currentRoom = null;
    state.selectedRoom = null;
    state.roomPlayers = [];
    saveTokens("", "");
    
    document.querySelector("#rooms-table tbody").innerHTML = "";
    setMsg("session-msg", "✅ Вы покинули базу", "success");
    logToUI("ℹ️ Вы вышли из боевой системы", "info");
    updateUIByAuthState();
    renderBattleInfo();
    updateFooterStats();
}

// ==== ROOMS ====

async function handleCreateRoom(e) {
    e.preventDefault();
    setMsg("create-room-msg", "🔄 Создание поля боя...", "info");

    if (!state.accessToken) {
        setMsg("create-room-msg", "❌ Для создания поля боя нужен доступ", "error");
        return;
    }

    if (state.currentRoom) {
        setMsg("create-room-msg", "❌ Сначала покиньте текущее поле боя", "error");
        return;
    }

    const fd = new FormData(e.target);
    const body = {
        name: fd.get("name").trim(),
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
        setMsg("create-room-msg", "✅ Поле боя создано!", "success");
        logToUI(`✅ Поле боя "${body.name}" готово к сражению`, "success");
        e.target.reset();
        
    } catch (err) {
        setMsg("create-room-msg", `❌ Ошибка: ${err.message}`, "error");
        logToUI(`❌ Ошибка создания поля боя: ${err.message}`, "error");
    }
}

async function loadRooms() {
    if (!state.accessToken) {
        setMsg("rooms-msg", "⚠️ Войдите для просмотра полей боя", "info");
        return;
    }

    try {
        const res = await api("/rooms", { headers: authHeaders() });
        const baseRooms = res.data || [];

        // Для каждой комнаты подгружаем реальный список игроков
        const roomsWithCounts = await Promise.all(
            baseRooms.map(async (room) => {
                try {
                    const r = await api(`/rooms/${room.id}/players`, {
                        headers: authHeaders()
                    });
                    const players = r.data?.players || [];
                    return {
                        ...room,
                        players_count: players.length
                    };
                } catch (e) {
                    // Если запрос упал — не ломаем всё, просто возвращаем как есть
                    return room;
                }
            })
        );

        state.rooms = roomsWithCounts;

        // Подсчёт игроков в бою и онлайна по всем комнатам
        state.battlingCount = state.rooms.reduce((total, room) => {
            return total + (room.players_count || 0);
        }, 0);
        state.onlineCount = state.battlingCount; // пусть пока это одно и то же

        const tbody = document.querySelector("#rooms-table tbody");
        tbody.innerHTML = "";

        if (state.rooms.length === 0) {
            tbody.innerHTML = `
                <tr class="empty-row">
                    <td colspan="7" style="text-align: center; padding: 30px; color: #6a7b9c;">
                        <i class="fas fa-door-closed" style="font-size: 28px; margin-bottom: 10px;"></i>
                        <div style="font-size: 14px; font-weight: 600;">Нет активных полей боя</div>
                        <div style="font-size: 12px; margin-top: 8px;">Создайте первое поле боя!</div>
                    </td>
                </tr>
            `;
            setMsg("rooms-msg", "Нет активных полей боя", "info");
            updateFooterStats();
            return;
        }

        state.rooms.forEach(room => {
            const tr = document.createElement("tr");
            const playerCount = room.players_count || 0;
            const maxPlayers = room.max_players;
            const isFull = playerCount >= maxPlayers;
            const isCurrent = state.currentRoom === room.id;
            const isSelected = state.selectedRoom?.id === room.id;

            if (isSelected) tr.classList.add("room-selected");
            if (isCurrent) tr.classList.add("room-current");

            // Формат: 2/8 (игроки/макс)
            const playersText = `${playerCount}/${maxPlayers}`;

            tr.innerHTML = `
                <td><span class="room-id">#${room.id}</span></td>
                <td>
                    <div class="room-name">
                        <i class="fas ${isFull ? 'fa-lock' : 'fa-unlock'}"></i>
                        ${room.name}
                        ${isCurrent ? '<span class="current-room-badge"><i class="fas fa-tank"></i> Вы здесь</span>' : ''}
                    </div>
                </td>
                <td><span class="room-owner">${room.owner_nickname || room.owner_id?.slice(0, 8)}</span></td>
                <td>
                    <div class="players-count ${isFull ? 'full' : ''}">
                        <i class="fas fa-users"></i>
                        <span class="player-indicator ${playerCount === 0 ? 'empty' : isFull ? 'full' : 'active'}">
                            ${playersText}
                        </span>
                    </div>
                </td>
                <td><span class="room-time">${room.duration_minutes} мин</span></td>
                <td><span class="room-gold"><i class="fas fa-coins"></i> ${room.gold_per_kill}</span></td>
                <td><span class="room-fund">${room.fund_modifier}x</span></td>
            `;

            tr.addEventListener("click", () => {
                selectRoom(room, tr);
            });

            tbody.appendChild(tr);
        });

        setMsg("rooms-msg", `Полей боя: ${state.rooms.length}`, "success");
        updateFooterStats();
        
    } catch (err) {
        setMsg("rooms-msg", `❌ Ошибка загрузки: ${err.message}`, "error");
    }
}

function selectRoom(room, rowEl) {
    // Нельзя выбирать другую комнату, если уже в какой-то
    if (state.currentRoom && state.currentRoom !== room.id) {
        logToUI("❌ Вы не можете выбрать другое поле боя, пока находитесь в текущем", "error");
        return;
    }

    state.selectedRoom = room;

    // Снимаем выделение со всех строк
    document.querySelectorAll("#rooms-table tbody tr").forEach(tr => {
        tr.classList.remove("room-selected");
    });

    // Подсвечиваем выбранную строку, если она передана
    if (rowEl) {
        rowEl.classList.add("room-selected");
    }

    // Подгружаем игроков и обновляем правую панель + кнопку
    loadRoomPlayers(room);
}

// ==== ROOM MANAGEMENT ====

async function joinSelectedRoom() {
    if (!state.selectedRoom) {
        logToUI("❌ Выберите поле боя для присоединения", "error");
        return;
    }

    if (state.currentRoom) {
        logToUI("❌ Вы уже находитесь на поле боя", "error");
        return;
    }

    const room = state.selectedRoom;
    const playerCount = state.roomPlayers.length;
    const maxPlayers = room.max_players;

    if (playerCount >= maxPlayers) {
        logToUI("❌ Поле боя заполнено", "error");
        return;
    }

    try {
        logToUI(`🔄 Вход на поле боя "${room.name}"...`, "info");
        
        await api(`/rooms/${room.id}/join`, {
            method: "POST",
            headers: authHeaders()
        });

        state.currentRoom = room.id;
        logToUI(`✅ Вы вступили на поле боя "${room.name}"`, "success");
        await Promise.all([loadRooms(), loadRoomPlayers(room)]);
        updateUserStatus();
        
    } catch (err) {
        logToUI(`❌ Ошибка входа: ${err.message}`, "error");
    }
}

async function leaveCurrentRoom() {
    if (!state.currentRoom) {
        logToUI("❌ Вы не находитесь на поле боя", "error");
        return;
    }

    try {
        const room = state.rooms.find(r => r.id === state.currentRoom) || state.selectedRoom;
        const roomName = room?.name || "неизвестное";
        
        logToUI(`🔄 Выход с поля боя "${roomName}"...`, "info");
        
        await api(`/rooms/${state.currentRoom}/leave`, {
            method: "POST",
            headers: authHeaders()
        });

        state.currentRoom = null;
        logToUI(`Вы покинули поле боя "${roomName}"`, "info");
        await Promise.all([loadRooms(), loadRoomPlayers(room)]);
        updateUserStatus();
        
    } catch (err) {
        logToUI(`❌ Ошибка выхода: ${err.message}`, "error");
    }
}

async function handleJoinLeaveClick() {
    if (!state.accessToken) {
        logToUI("❌ Войдите в боевую систему", "error");
        return;
    }

    if (state.currentRoom) {
        await leaveCurrentRoom();
    } else {
        await joinSelectedRoom();
    }
}

// ==== ROOM PLAYERS ====

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

        const payload = res.data || {};
        state.roomPlayers = payload.players || [];
        renderBattleInfo();
        
    } catch (err) {
        state.roomPlayers = [];
        renderBattleInfo();
    }
}

function renderBattleInfo() {
    const titleEl = document.getElementById("battle-title");
    const modeEl = document.getElementById("battle-mode");
    const timeEl = document.getElementById("battle-time");
    const playersEl = document.getElementById("battle-players");
    const rewardEl = document.getElementById("battle-reward");
    const joinBtn = document.getElementById("join-room-btn");
    const listEl = document.getElementById("battle-players-list");
    const hintEl = document.getElementById("battle-hint");

    const room = state.selectedRoom;

    if (!room) {
        titleEl.textContent = "Выберите поле боя";
        modeEl.textContent = "Deathmatch";
        timeEl.textContent = "—";
        playersEl.textContent = "0/8";
        rewardEl.textContent = "0 gold";
        joinBtn.disabled = true;
        joinBtn.innerHTML = '<i class="fas fa-play"></i> <span>ВЫБЕРИТЕ ПОЛЕ БОЯ</span>';
        joinBtn.onclick = null;
        
        if (hintEl) {
            hintEl.textContent = "Выберите поле боя из списка слева";
        }
        
        listEl.innerHTML = `
            <li class="empty">
                <i class="fas fa-door-open"></i>
                Выберите поле боя из списка слева
            </li>
        `;
        return;
    }

    const playerCount = state.roomPlayers.length;
    const maxPlayers = room.max_players;
    const isFull = playerCount >= maxPlayers;
    const isInThisRoom = state.currentRoom === room.id;

    titleEl.textContent = room.name;
    modeEl.textContent = "Deathmatch";
    timeEl.textContent = `${room.duration_minutes} мин`;
    playersEl.textContent = `${playerCount}/${maxPlayers}`;
    rewardEl.textContent = `${room.gold_per_kill * room.fund_modifier} gold`;

    // Обновляем кнопку в правой панели
    if (isInThisRoom) {
        joinBtn.disabled = false;
        joinBtn.innerHTML = '<i class="fas fa-sign-out-alt"></i> <span>ПОКИНУТЬ ПОЛЕ БОЯ</span>';
        joinBtn.classList.remove("btn-green", "btn-blue");
        joinBtn.classList.add("btn-red");
        joinBtn.onclick = handleJoinLeaveClick;
        
        if (hintEl) {
            hintEl.textContent = "Вы уже находитесь на этом поле боя";
        }
    } else if (state.currentRoom) {
        joinBtn.disabled = true;
        joinBtn.innerHTML = '<i class="fas fa-ban"></i> <span>ВЫ УЖЕ В БОЮ</span>';
        joinBtn.classList.remove("btn-green", "btn-red");
        joinBtn.classList.add("btn-blue");
        joinBtn.onclick = null;
        
        if (hintEl) {
            hintEl.textContent = "Покиньте текущее поле боя, чтобы присоединиться к другому";
        }
    } else if (isFull) {
        joinBtn.disabled = true;
        joinBtn.innerHTML = '<i class="fas fa-lock"></i> <span>ПОЛЕ БОЯ ЗАПОЛНЕНО</span>';
        joinBtn.classList.remove("btn-red", "btn-blue");
        joinBtn.classList.add("btn-green");
        joinBtn.onclick = null;
        
        if (hintEl) {
            hintEl.textContent = "Дождитесь, когда освободится место";
        }
    } else {
        joinBtn.disabled = false;
        joinBtn.innerHTML = '<i class="fas fa-play"></i> <span>ВСТУПИТЬ В БИТВУ</span>';
        joinBtn.classList.remove("btn-red", "btn-blue");
        joinBtn.classList.add("btn-green");
        joinBtn.onclick = handleJoinLeaveClick;
        
        if (hintEl) {
            hintEl.textContent = "Нажмите для присоединения к битве";
        }
    }

    // Список игроков
    listEl.innerHTML = "";
    
    if (playerCount === 0) {
        listEl.innerHTML = `
            <li class="empty">
                <i class="fas fa-user-slash"></i>
                На поле боя пока никого
            </li>
        `;
    } else {
        state.roomPlayers.forEach((player, index) => {
            const li = document.createElement("li");
            const isOwner = player.id === room.owner_id;
            const isCurrentUser = state.currentUser?.id === player.id;
            
            li.className = isCurrentUser ? "current-player" : "";
            li.innerHTML = `
                <span class="player-rank">${index + 1}.</span>
                <span class="player-name">${player.nickname}</span>
                ${isOwner ? '<span class="player-owner"><i class="fas fa-crown"></i> Командир</span>' : ''}
                ${isCurrentUser ? '<span class="player-you"><i class="fas fa-user"></i> Вы</span>' : ''}
            `;
            listEl.appendChild(li);
        });
    }
}

// ==== INIT ====
document.addEventListener("DOMContentLoaded", async () => {
    // Загружаем токены
    loadTokens();
    renderSession();
    updateServerStatus();
    
    // Event Listeners
    document.getElementById("login-form").addEventListener("submit", handleLogin);
    document.getElementById("register-form").addEventListener("submit", handleRegister);
    document.getElementById("refresh-btn").addEventListener("click", handleRefresh);
    document.getElementById("logout-btn").addEventListener("click", handleLogout);
    document.getElementById("create-room-form").addEventListener("submit", handleCreateRoom);
    document.getElementById("reload-rooms-btn").addEventListener("click", loadRooms);
    document.getElementById("clear-log-btn").addEventListener("click", () => {
        document.getElementById("fake-chat").innerHTML = `
            <div class="log-entry info"><span class="time">[${new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}]</span> Система: Лог очищен</div>
        `;
        logToUI("🔄 Боевой лог очищен", "info");
    });
    
    // Мобильное меню
    document.getElementById("mobile-menu-btn").addEventListener("click", toggleMobileMenu);
    
    // Кнопка в правой панели
    const joinBtn = document.getElementById("join-room-btn");
    if (joinBtn) {
        joinBtn.addEventListener("click", handleJoinLeaveClick);
    }
    
    if (state.accessToken) {
        try {
            await fetchMe();
            await loadRooms();
            updateUIByAuthState();
            logToUI("✅ Боевая сессия восстановлена", "success");
        } catch (err) {
            logToUI("❌ Ошибка загрузки боевых данных", "error");
            handleLogout();
        }
    }
    
    renderBattleInfo();
    
    // Автообновление каждые 5 секунд
    state.pollingInterval = setInterval(async () => {
        if (state.accessToken && state.autoRefresh) {
            try {
                await loadRooms();
                if (state.selectedRoom) {
                    await loadRoomPlayers(state.selectedRoom);
                }
            } catch (err) {
                // Тихий фейл, не логируем
            }
        }
    }, 5000);
    
    // Анимация приветствия
    setTimeout(() => {
        logToUI("🎮 Интерфейс Tanki Online загружен", "success");
        logToUI("✅ Боевая система готова к работе", "success");
        logToUI("💥 Добро пожаловать на поле боя, танкист!", "info");
    }, 1000);
    
    // Адаптив для изменения размера окна
    window.addEventListener("resize", () => {
        if (window.innerWidth > 767 && state.mobileMenuOpen) {
            toggleMobileMenu();
        }
    });
});

async function fetchMe() {
    if (!state.accessToken) return;

    try {
        const data = await api("/me", {
            headers: authHeaders()
        });
        state.currentUser = data.data;
        renderSession();
        updateUIByAuthState();
        
    } catch (err) {
        state.currentUser = null;
        logToUI("❌ Ошибка загрузки боевого профиля", "error");
    }
}

window.addEventListener("beforeunload", () => {
    const roomId = state.currentRoom;
    const token = state.accessToken;

    // если не авторизован или не в комнате — ничего не делаем
    if (!roomId || !token) return;

    try {
        fetch(`/api/v1/rooms/${roomId}/leave`, {
            method: "POST",
            keepalive: true, // чтобы запрос не сдох при закрытии вкладки
            headers: {
                "Content-Type": "application/json",
                "Authorization": `Bearer ${token}`,
            },
            body: JSON.stringify({ reason: "tab_closed" })
        });
    } catch (e) {
        // тут пофиг, вкладка всё равно умирает
    }
});









// ===========================
// ==== GAME / WS / MAP ======
// ===========================

// runtime-состояние боя (отдельно от state, чтобы не засорять)
let gameWS = null;
const gameInput = {
  forward: false,
  backward: false,
  rotate_left: false,
  rotate_right: false,
  shoot: false,
};

// Подключение к WS /ws/game
function connectGameWS() {
  if (!state.currentRoom || !state.accessToken) {
    return;
  }

  // Если старый WS висит — убиваем
  if (gameWS) {
    try { gameWS.close(); } catch (e) {}
    gameWS = null;
  }

  const proto = location.protocol === "https:" ? "wss" : "ws";
  const url = `${proto}://${location.host}/ws/game?room_id=${state.currentRoom}&token=${state.accessToken}`;

  logToUI("🔌 Подключение к полю боя...", "info");

  gameWS = new WebSocket(url);

  gameWS.onopen = () => {
    logToUI("✅ Соединение с полем боя установлено", "success");
    sendGameInput(); // сразу отправим нулевой инпут
  };

  gameWS.onclose = () => {
    logToUI("⚠️ Соединение с полем боя закрыто", "error");
    gameWS = null;
  };

  gameWS.onerror = (e) => {
    console.log("WS GAME ERROR", e);
    logToUI("❌ Ошибка WebSocket боя", "error");
  };

  gameWS.onmessage = handleGameMessage;
}

function disconnectGameWS() {
  if (gameWS) {
    try { gameWS.close(); } catch (e) {}
    gameWS = null;
  }
}

// Обработка сообщений от сервера
function handleGameMessage(evt) {
  let msg;
  try {
    msg = JSON.parse(evt.data);
  } catch {
    return;
  }

  const type = msg.type || msg.Type;
  if (type === "state") {
    const data = msg.data || msg.Data || msg.state || msg.State;
    if (data) {
      renderBattleCanvas(data);
    }
  }
}

// Отправка инпута на сервер
function sendGameInput() {
  if (!gameWS || gameWS.readyState !== WebSocket.OPEN) return;

  gameWS.send(JSON.stringify({
    type: "input",
    input: { ...gameInput },
  }));
}

// Обновление инпута по клавишам
function updateInputKey(code, pressed) {
  switch (code) {
    case "KeyW":
    case "ArrowUp":
      gameInput.forward = pressed;
      break;
    case "KeyS":
    case "ArrowDown":
      gameInput.backward = pressed;
      break;
    case "KeyA":
    case "ArrowLeft":
      gameInput.rotate_left = pressed;
      break;
    case "KeyD":
    case "ArrowRight":
      gameInput.rotate_right = pressed;
      break;
    case "Space":
      gameInput.shoot = pressed;
      break;
    default:
      return; // не слать лишний раз
  }
  sendGameInput();
}

// Навешиваем глобальные слушатели клавиатуры (кроме инпутов)
window.addEventListener("keydown", (e) => {
  const tag = (e.target && e.target.tagName) || "";
  if (tag === "INPUT" || tag === "TEXTAREA") return;
  if (e.repeat) return;
  updateInputKey(e.code, true);
});

window.addEventListener("keyup", (e) => {
  const tag = (e.target && e.target.tagName) || "";
  if (tag === "INPUT" || tag === "TEXTAREA") return;
  updateInputKey(e.code, false);
});

// ============================
// ====== РЕНДЕР КАРТЫ ========
// ============================

const ARENA_W = 1000;
const ARENA_H = 1000;

function renderBattleCanvas(snap) {
  const canvas = document.getElementById("battle-canvas");
  if (!canvas) return;

  const ctx = canvas.getContext("2d");
  const w = canvas.width;
  const h = canvas.height;
  const scaleX = w / ARENA_W;
  const scaleY = h / ARENA_H;

  // фон
  ctx.fillStyle = "#05070d";
  ctx.fillRect(0, 0, w, h);

  // лёгкая сетка
  ctx.strokeStyle = "#151b26";
  ctx.lineWidth = 1;
  const gridStep = 100;
  for (let x = 0; x <= ARENA_W; x += gridStep) {
    ctx.beginPath();
    ctx.moveTo(x * scaleX, 0);
    ctx.lineTo(x * scaleX, h);
    ctx.stroke();
  }
  for (let y = 0; y <= ARENA_H; y += gridStep) {
    ctx.beginPath();
    ctx.moveTo(0, y * scaleY);
    ctx.lineTo(w, y * scaleY);
    ctx.stroke();
  }

  const players = snap.players || snap.Players || [];
  const bullets = snap.bullets || snap.Bullets || [];

  const myId = state.currentUser?.id || state.currentUser?.user_id || null;

  // Пули
  bullets.forEach((b) => {
    const x = b.x || b.X || 0;
    const y = b.y || b.Y || 0;

    ctx.beginPath();
    ctx.arc(x * scaleX, y * scaleY, 3, 0, Math.PI * 2);
    ctx.fillStyle = "#ffb84a";
    ctx.fill();
  });

  // Танки
  players.forEach((p) => {
    const id = p.id || p.ID;
    const x = p.x || p.X || 0;
    const y = p.y || p.Y || 0;
    const angle = p.angle || p.Angle || 0;
    const hp = p.hp ?? p.HP ?? 0;

    const screenX = x * scaleX;
    const screenY = y * scaleY;

    const isMe = myId && String(id) === String(myId);

    const bodyWidth = 24;
    const bodyHeight = 32;

    // тело
    ctx.save();
    ctx.translate(screenX, screenY);
    ctx.rotate(angle);

    ctx.fillStyle = isMe ? "#4cd137" : "#95a5a6";
    ctx.fillRect(-bodyWidth / 2, -bodyHeight / 2, bodyWidth, bodyHeight);

    // башня/ствол
    ctx.fillStyle = isMe ? "#dcdde1" : "#ecf0f1";
    ctx.fillRect(-4, -bodyHeight / 2 - 8, 8, 16);

    ctx.restore();

    // хп-бар
    const hpPerc = Math.max(0, Math.min(1, hp / 100));
    const barW = 30;
    const barH = 4;

    ctx.fillStyle = "#2c3e50";
    ctx.fillRect(screenX - barW / 2, screenY - bodyHeight, barW, barH);

    ctx.fillStyle = hpPerc > 0.5 ? "#2ecc71" : "#e74c3c";
    ctx.fillRect(screenX - barW / 2, screenY - bodyHeight, barW * hpPerc, barH);
  });

  // рамка
  ctx.strokeStyle = "#2f3640";
  ctx.lineWidth = 2;
  ctx.strokeRect(0, 0, w, h);
}

// ============================
// ПЕРЕОПРЕДЕЛЯЕМ join/leave,
// чтобы поднимать/убивать WS
// ============================

// joinSelectedRoom с подключением WS
async function joinSelectedRoom() {
  if (!state.selectedRoom) {
    logToUI("❌ Выберите поле боя для присоединения", "error");
    return;
  }
  if (state.currentRoom) {
    logToUI("❌ Вы уже находитесь на поле боя", "error");
    return;
  }
  const room = state.selectedRoom;
  const playerCount = state.roomPlayers.length;
  const maxPlayers = room.max_players;

  if (playerCount >= maxPlayers) {
    logToUI("❌ Поле боя заполнено", "error");
    return;
  }

  try {
    logToUI(` Вход на поле боя "${room.name}"...`, "info");
    await api(`/rooms/${room.id}/join`, { method: "POST", headers: authHeaders() });
    state.currentRoom = room.id;
    logToUI(`✅ Вы вступили на поле боя "${room.name}"`, "success");

    await Promise.all([loadRooms(), loadRoomPlayers(room)]);
    updateUserStatus();

    // 🔌 после успешного join — подключаем WS
    connectGameWS();
  } catch (err) {
    logToUI(`❌ Ошибка входа: ${err.message}`, "error");
  }
}

// leaveCurrentRoom с отключением WS
async function leaveCurrentRoom() {
  if (!state.currentRoom) {
    logToUI("❌ Вы не находитесь на поле боя", "error");
    return;
  }

  try {
    const room = state.rooms.find(r => r.id === state.currentRoom) || state.selectedRoom;
    const roomName = room?.name || "неизвестное";

    logToUI(` Выход с поля боя "${roomName}"...`, "info");
    await api(`/rooms/${state.currentRoom}/leave`, { method: "POST", headers: authHeaders() });

    state.currentRoom = null;
    logToUI(`Вы покинули поле боя "${roomName}"`, "info");

    await Promise.all([loadRooms(), loadRoomPlayers(room)]);
    updateUserStatus();

    // ❌ рубим WS
    disconnectGameWS();
  } catch (err) {
    logToUI(`❌ Ошибка выхода: ${err.message}`, "error");
  }
}
