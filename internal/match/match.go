package match

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sundabaoh-rgb/tankionline/internal/domain"
	"github.com/sundabaoh-rgb/tankionline/internal/logger"
	"github.com/sundabaoh-rgb/tankionline/internal/ws"
)

// ===== ГЛОБАЛЬНЫЕ КОНСТАНТЫ МАТЧА =====

const (
	maxHP        = 100.0 // ХП танка
	respawnDelay = 3.0   // через сколько секунд после смерти респавним
	playerRadius = 0.75  // радиус хитбокса танка в игровых координатах
	bulletDamage = 15.0  // урон пули
	turretNoAim  = 999999.0
)

// как игрок выглядит в state
type PlayerState struct {
	UserID uuid.UUID
	X      float64
	Y      float64

	VelX float64
	VelY float64

	Angle float64 // <-- угол в радианах, куда смотрит ствол

	// перезарядка между выстрелами, в секундах
	ShootCooldown float64
	HP            float64

	TurretAngle float64 // угол башни в градусах

	Kills  int
	Deaths int

	Dead         bool    // сейчас мёртв?
	RespawnTimer float64 // сколько осталось до респавна
}

// что мы храним про нажатые кнопки
type InputState struct {
	Up          bool
	Down        bool
	Left        bool
	Right       bool
	Shoot       bool
	TurretLeft  bool
	TurretRight bool
	TurretTo    float64
}

// то, что отсылаем клиенту
type PlayerView struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	HP float64 `json:"hp"`

	Kills       int     `json:"kills"`
	Deaths      int     `json:"deaths"`
	TurretAngle float64 `json:"turret_angle"`

	Dead bool `json:"dead,omitempty"`

	ShootCooldown    float64 `json:"shoot_cooldown"`     // сколько сек осталось
	MaxShootCooldown float64 `json:"max_shoot_cooldown"` // константа fireCooldown

	Angle float64 `json:"angle"` // <-- ГРАДУСЫ для клиента
}

// внутренняя пуля
type BulletState struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	X       float64
	Y       float64
	VelX    float64
	VelY    float64
	TTL     float64 // сколько секунд ещё живёт
}

// то, что летит клиенту
type BulletView struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

type StateSnapshot struct {
	Players []PlayerView `json:"players"`
	Bullets []BulletView `json:"bullets,omitempty"`
}

// лог смерти в рамках матча
type DeathEvent struct {
	KillerID uuid.UUID `json:"killer_id"`
	VictimID uuid.UUID `json:"victim_id"`
	Time     time.Time `json:"time"`
}

// один матч (одна битва)
type Match struct {
	BattleID uuid.UUID

	log logger.Logger
	hub *ws.Hub

	mu      sync.Mutex
	players map[uuid.UUID]*PlayerState
	inputs  map[uuid.UUID]InputState
	bullets []*BulletState

	tick time.Duration

	// для ротации спавнов
	spawnIdx int

	// лог смертей внутри матча
	deaths []DeathEvent
}

// создаётся из MatchService
func NewMatch(battleID uuid.UUID, hub *ws.Hub, log logger.Logger) *Match {
	return &Match{
		BattleID: battleID,
		log:      log.Named("match").With("battle_id", battleID.String()),
		hub:      hub,
		players:  make(map[uuid.UUID]*PlayerState),
		inputs:   make(map[uuid.UUID]InputState),
		bullets:  make([]*BulletState, 0),
		tick:     50 * time.Millisecond,
	}

}

func (m *Match) Run(ctx context.Context) {
	ticker := time.NewTicker(m.tick)
	defer ticker.Stop()

	last := time.Now()

	for {
		select {
		case <-ctx.Done():
			m.log.Info("match stopped")
			return
		case now := <-ticker.C:
			dt := now.Sub(last).Seconds()
			last = now
			m.step(dt)
		}
	}
}

// --- хелпер спавна игрока ---
func (m *Match) spawnPlayer(p *PlayerState) {
	spawns := [][2]float64{
		{2, 2},
		{18, 2},
		{2, 18},
		{18, 18},
	}

	if len(spawns) == 0 {
		p.X, p.Y = 1, 1
	} else {
		if m.spawnIdx >= len(spawns) {
			m.spawnIdx = 0
		}
		pos := spawns[m.spawnIdx]
		m.spawnIdx++
		p.X, p.Y = pos[0], pos[1]
	}

	p.HP = maxHP
	p.ShootCooldown = 0
	p.Dead = false
	p.RespawnTimer = 0

	// корпус и башня смотрят вперёд (вверх/вправо — на твой вкус)
	p.Angle = -math.Pi / 2 // радианы
	p.TurretAngle = -90    // градусы, синхронно с корпусом
	p.VelX, p.VelY = 0, 0
}

// добавить игрока в матч
func (m *Match) AddPlayer(userID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.players[userID]; ok {
		return
	}

	ps := &PlayerState{
		UserID: userID,
	}
	m.spawnPlayer(ps)

	m.players[userID] = ps
	m.log.Info("player joined match", "user_id", userID.String())
}

// убрать игрока
func (m *Match) RemovePlayer(userID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.players, userID)
	delete(m.inputs, userID)
	m.log.Info("player left match", "user_id", userID.String())
}

// выставить последнее нажатое игроком
func (m *Match) SetInput(userID uuid.UUID, in domain.PlayerInput) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.inputs[userID] = InputState{
		Up:          in.Up,
		Down:        in.Down,
		Left:        in.Left,
		Right:       in.Right,
		Shoot:       in.Shoot,
		TurretLeft:  in.TurretLeft,
		TurretRight: in.TurretRight,
		TurretTo:    in.TurretTo,
	}
}

// обработка смерти (урон добил)
// обработка смерти (урон добил)
func (m *Match) handleDeath(killerID, victimID uuid.UUID) {
	p, ok := m.players[victimID]
	if !ok {
		return
	}
	if p.Dead {
		return
	}

	p.Dead = true
	p.HP = 0
	p.RespawnTimer = respawnDelay
	p.Deaths++ // <-- смерть жертве

	if killer, ok := m.players[killerID]; ok {
		if killerID != victimID { // на всякий случай отфильтровать суицид
			killer.Kills++ // <-- килл киллеру
		}
	}

	ev := DeathEvent{
		KillerID: killerID,
		VictimID: victimID,
		Time:     time.Now(),
	}
	m.deaths = append(m.deaths, ev)

	m.log.Info("player died",
		"victim", victimID.String(),
		"killer", killerID.String(),
	)

	// простое сообщение на клиент — killfeed
	payload, _ := json.Marshal(ws.ServerMessage{
		Type: "death",
		Data: map[string]string{
			"killer_id": killerID.String(),
			"victim_id": victimID.String(),
		},
	})
	m.hub.BroadcastState(m.BattleID, payload)
}

// один тик игры
func (m *Match) step(dt float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	const (
		speed        = 4.0 // скорость танка, одинаковая по X/Y
		bulletSpeed  = 16.0
		bulletTTL    = 2.0
		fireCooldown = 0.9
		fieldMin     = 0.0
		fieldMax     = 20.0
	)

	// --- игроки ---
	for id, p := range m.players {
		in := m.inputs[id]

		// если мёртв — только респавн таймер
		if p.Dead {
			p.RespawnTimer -= dt
			if p.RespawnTimer <= 0 {
				m.spawnPlayer(p)
				m.log.Info("player respawned", "user_id", id.String())
			}
			continue
		}

		// --- башня ---
		if in.TurretLeft {
			p.TurretAngle -= 180 * dt
		}
		if in.TurretRight {
			p.TurretAngle += 180 * dt
		}

		// мышь
		if in.TurretTo != turretNoAim {
			p.TurretAngle = in.TurretTo
		}

		// нормализуем к 0..360
		for p.TurretAngle < 0 {
			p.TurretAngle += 360
		}
		for p.TurretAngle >= 360 {
			p.TurretAngle -= 360
		}

		// --- кулдаун ---
		if p.ShootCooldown > 0 {
			p.ShootCooldown -= dt
			if p.ShootCooldown < 0 {
				p.ShootCooldown = 0
			}
		}

		// --- движение корпуса ---
		vx, vy := 0.0, 0.0
		if in.Up {
			vy -= 1
		}
		if in.Down {
			vy += 1
		}
		if in.Left {
			vx -= 1
		}
		if in.Right {
			vx += 1
		}

		if vx != 0 || vy != 0 {
			length := math.Hypot(vx, vy)
			vx /= length
			vy /= length

			p.VelX = vx * speed
			p.VelY = vy * speed * 2

			p.X += p.VelX * dt
			p.Y += p.VelY * dt

			// корпус смотрит туда, куда двигаемся
			p.Angle = math.Atan2(vy, vx)
		} else {
			// легкое затухание
			p.VelX *= 0.85
			p.VelY *= 0.85
			p.X += p.VelX * dt
			p.Y += p.VelY * dt
		}

		// границы
		if p.X < fieldMin {
			p.X = fieldMin
		}
		if p.X > fieldMax {
			p.X = fieldMax
		}
		if p.Y < fieldMin {
			p.Y = fieldMin
		}
		if p.Y > fieldMax {
			p.Y = fieldMax
		}

		// --- выстрел: ИЛИ по корпусу, ИЛИ по башне ---
		if in.Shoot && p.ShootCooldown == 0 {
			ang := p.TurretAngle * math.Pi / 180
			dirX := math.Cos(ang)
			dirY := math.Sin(ang)

			if dirX == 0 && dirY == 0 {
				dirX, dirY = 0, -1 // дефолт: вверх
			}

			b := &BulletState{
				ID:      uuid.New(),
				OwnerID: id,
				X:       p.X + dirX*0.6,
				Y:       p.Y + dirY*0.9,
				VelX:    dirX * bulletSpeed,
				VelY:    dirY * bulletSpeed,
				TTL:     bulletTTL,
			}
			m.bullets = append(m.bullets, b)

			p.ShootCooldown = fireCooldown

			//rofl
		}
	}

	// --- обновляем пули ---
	if len(m.bullets) > 0 {
		alive := m.bullets[:0]

		for _, b := range m.bullets {
			b.X += b.VelX * dt
			b.Y += b.VelY * dt
			b.TTL -= dt

			// если уже умерла по времени — дальше даже не проверяем
			if b.TTL <= 0 {
				continue
			}
			// вышла за поле
			if b.X < fieldMin || b.X > fieldMax || b.Y < fieldMin || b.Y > fieldMax {
				continue
			}

			// проверяем коллизию с игроками
			hit := false
			for pid, p := range m.players {
				if pid == b.OwnerID {
					continue // себя не бьём
				}
				if p.Dead {
					continue
				}

				dx := b.X - p.X
				dy := b.Y - p.Y
				if dx*dx+dy*dy <= playerRadius*playerRadius {
					p.HP -= bulletDamage
					if p.HP <= 0 {
						p.HP = 0
						m.handleDeath(b.OwnerID, p.UserID)
					}
					hit = true
					break
				}

			}

			if hit {
				// не добавляем пулю в alive — она "разорвалась"
				continue
			}

			alive = append(alive, b)
		}

		m.bullets = alive
	}

	// --- собираем снапшот ---
	snap := StateSnapshot{
		Players: make([]PlayerView, 0, len(m.players)),
		Bullets: make([]BulletView, 0, len(m.bullets)),
	}

	for _, p := range m.players {
		angleDeg := p.Angle * 180.0 / math.Pi

		snap.Players = append(snap.Players, PlayerView{
			ID:               p.UserID.String(),
			X:                p.X,
			Y:                p.Y,
			HP:               p.HP,
			Kills:            p.Kills,
			Deaths:           p.Deaths,
			Dead:             p.Dead,
			ShootCooldown:    p.ShootCooldown,
			MaxShootCooldown: fireCooldown,
			Angle:            angleDeg,
			TurretAngle:      p.TurretAngle,
		})
	}

	for _, b := range m.bullets {
		snap.Bullets = append(snap.Bullets, BulletView{
			ID: b.ID.String(),
			X:  b.X,
			Y:  b.Y,
		})
	}

	if len(snap.Players) == 0 && len(snap.Bullets) == 0 {
		return
	}

	payload, _ := json.Marshal(ws.ServerMessage{
		Type: "state",
		Data: snap,
	})

	m.hub.BroadcastState(m.BattleID, payload)
}
