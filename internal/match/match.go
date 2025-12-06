package match

import (
	"context"
	"encoding/json"
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
	VelX   float64
	VelY   float64
}

// что мы храним про нажатые кнопки
type InputState struct {
	Up    bool
	Down  bool
	Left  bool
	Right bool
}

// то, что отсылаем клиенту
type PlayerView struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

type StateSnapshot struct {
	Players []PlayerView `json:"players"`
}

// один матч (одна битва)
type Match struct {
	BattleID uuid.UUID

	log logger.Logger
	hub *ws.Hub

	mu      sync.Mutex
	players map[uuid.UUID]*PlayerState
	inputs  map[uuid.UUID]InputState

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
		tick:     50 * time.Millisecond, // 20 тиков в секунду
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
	}
}

// один тик игры
func (m *Match) step(dt float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	const speed = 5.0
	const fieldMin = 0.0
	const fieldMax = 20.0

	for id, p := range m.players {
		in := m.inputs[id]

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

		p.X += vx * speed * dt
		p.Y += vy * speed * dt

		// простые границы
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
	}

	// собираем снапшот
	snap := StateSnapshot{
		Players: make([]PlayerView, 0, len(m.players)),
	}
	for _, p := range m.players {
		snap.Players = append(snap.Players, PlayerView{
			ID: p.UserID.String(),
			X:  p.X,
			Y:  p.Y,
		})
	}

	// кодируем и рассылаем через Hub
	if len(snap.Players) == 0 {
		return
	}

	payload, _ := json.Marshal(ws.ServerMessage{
		Type: "state",
		Data: snap,
	})

	m.hub.BroadcastState(m.BattleID, payload)
}
