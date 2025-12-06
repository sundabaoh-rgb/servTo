package ws

import (
	"sync"

	"github.com/google/uuid"
	"github.com/sundabaoh-rgb/tankionline/internal/logger"
)

type Broadcast struct {
	BattleID uuid.UUID
	Data     []byte
}

type Hub struct {
	log logger.Logger

	mu sync.RWMutex
	// battleID -> set of connections
	battles map[uuid.UUID]map[*Connection]struct{}

	register   chan *Connection
	unregister chan *Connection
	broadcast  chan Broadcast
}

func NewHub(log logger.Logger) *Hub {
	return &Hub{
		log:        log.Named("ws_hub"),
		battles:    make(map[uuid.UUID]map[*Connection]struct{}),
		register:   make(chan *Connection),
		unregister: make(chan *Connection),
		broadcast:  make(chan Broadcast, 128),
	}
}

func (h *Hub) Run() {
	h.log.Info("ws hub started")
	for {
		select {
		case c := <-h.register:
			h.addConn(c)
		case c := <-h.unregister:
			h.removeConn(c)
		case msg := <-h.broadcast:
			h.sendToBattle(msg)
		}
	}
}

func (h *Hub) addConn(c *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns, ok := h.battles[c.battleID]
	if !ok {
		conns = make(map[*Connection]struct{})
		h.battles[c.battleID] = conns
	}
	conns[c] = struct{}{}

	h.log.Info("ws registered connection",
		"battle_id", c.battleID,
		"user_id", c.userID,
		"total_conns", len(conns),
	)
}

func (h *Hub) removeConn(c *Connection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns, ok := h.battles[c.battleID]
	if !ok {
		return
	}
	delete(conns, c)
	if len(conns) == 0 {
		delete(h.battles, c.battleID)
	}
	h.log.Info("ws unregistered connection",
		"battle_id", c.battleID,
		"user_id", c.userID,
	)
}

func (h *Hub) sendToBattle(msg Broadcast) {
	h.mu.RLock()
	conns := h.battles[msg.BattleID]
	h.mu.RUnlock()

	for c := range conns {
		select {
		case c.send <- msg.Data:
		default:
			// канал забился — выпиливаем коннект
			go func(c *Connection) {
				h.unregister <- c
				c.Close()
			}(c)
		}
	}
}

func (h *Hub) Register(c *Connection) {
	h.register <- c
}

func (h *Hub) BroadcastState(battleID uuid.UUID, data []byte) {
	h.broadcast <- Broadcast{
		BattleID: battleID,
		Data:     data,
	}
}
