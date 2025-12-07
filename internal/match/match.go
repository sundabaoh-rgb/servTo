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

// как игрок выглядит в state
type PlayerState struct {
	UserID uuid.UUID
	X      float64
	Y      float64

	VelX float64
	VelY float64

	// перезарядка между выстрелами, в секундах
	ShootCooldown float64
}

// что мы храним про нажатые кнопки
type InputState struct {
	Up    bool
	Down  bool
	Left  bool
	Right bool
	Shoot bool
}

// то, что отсылаем клиенту
type PlayerView struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
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

// добавить игрока в матч
func (m *Match) AddPlayer(userID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.players[userID]; ok {
		return
	}

	// пока просто рандомная позиция по кругу
	x := float64(len(m.players)*2 + 1)
	y := 1.0

	m.players[userID] = &PlayerState{
		UserID: userID,
		X:      x,
		Y:      y,
	}
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
		Up:    in.Up,
		Down:  in.Down,
		Left:  in.Left,
		Right: in.Right,
		Shoot: in.Shoot,
	}
}

// один тик игры
func (m *Match) step(dt float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	const (
		speed        = 3.0  // скорость танка (клеток в секунду)
		bulletSpeed  = 15.0 // скорость пули
		bulletTTL    = 2.0  // сколько живёт пуля в секундах
		fireCooldown = 0.9  // задержка между выстрелами
		fieldMin     = 0.0
		fieldMax     = 20.0
	)

	// --- обновляем игроков и спавним пули ---
	for id, p := range m.players {
		in := m.inputs[id]

		// кулдаун выстрела
		if p.ShootCooldown > 0 {
			p.ShootCooldown -= dt
			if p.ShootCooldown < 0 {
				p.ShootCooldown = 0
			}
		}

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

		// нормализуем направление
		if vx != 0 || vy != 0 {
			length := math.Hypot(vx, vy)
			vx /= length
			vy /= length

			// запоминаем скорость (если хочешь где-то использовать)
			p.VelX = vx * speed
			p.VelY = vy * (speed * 2)

			p.X += p.VelX * dt
			p.Y += p.VelY * dt
		}

		// границы поля
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

		// --- выстрел ---
		// --- выстрел ---
		if in.Shoot && p.ShootCooldown == 0 {
			// определяем направление выстрела
			dirX, dirY := 0.0, 0.0

			// 1) если сейчас жмём направление — стреляем туда
			if vx != 0 || vy != 0 {
				dirX, dirY = vx, vy
			} else if p.VelX != 0 || p.VelY != 0 {
				// 2) иначе — в сторону последнего движения
				length := math.Hypot(p.VelX, p.VelY)
				if length != 0 {
					dirX = p.VelX / length
					dirY = p.VelY / length
				}
			} else {
				// 3) вообще никогда не двигался — по дефолту вверх
				dirX = 0
				dirY = -1
			}

			// спавним пулю чуть впереди танка
			b := &BulletState{
				ID:      uuid.New(),
				OwnerID: id,
				X:       p.X + dirX*0.5,
				Y:       p.Y + dirY*0.5,
				VelX:    dirX * bulletSpeed,
				VelY:    dirY * bulletSpeed,
				TTL:     bulletTTL,
			}
			m.bullets = append(m.bullets, b)

			p.ShootCooldown = fireCooldown
		}

	}

	// --- обновляем пули ---
	if len(m.bullets) > 0 {
		alive := m.bullets[:0]

		for _, b := range m.bullets {
			b.X += b.VelX * dt
			b.Y += b.VelY * dt
			b.TTL -= dt

			// умерла — не оставляем
			if b.TTL <= 0 {
				continue
			}
			if b.X < fieldMin || b.X > fieldMax || b.Y < fieldMin || b.Y > fieldMax {
				continue
			}

			// TODO: тут потом будут проверки попаданий
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
		snap.Players = append(snap.Players, PlayerView{
			ID: p.UserID.String(),
			X:  p.X,
			Y:  p.Y,
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
