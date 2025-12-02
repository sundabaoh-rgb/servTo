package ws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
)

const (
	writeWait  = 10 * time.Second
	pingPeriod = 30 * time.Second
)

// ServerMessage — то, что сервер шлёт клиенту
type ServerMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// ClientMessage — то, что клиент шлёт серверу
type ClientMessage struct {
	Type  string              `json:"type"`
	Input *domain.PlayerInput `json:"input,omitempty"`
}

type Connection struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	battleID uuid.UUID
	userID   uuid.UUID
}

func NewConnection(h *Hub, wsConn *websocket.Conn, battleID, userID uuid.UUID) *Connection {
	return &Connection{
		hub:      h,
		conn:     wsConn,
		send:     make(chan []byte, 256),
		battleID: battleID,
		userID:   userID,
	}
}

func (c *Connection) Close() {
	_ = c.conn.Close()
}

// Запуск двух горутин: читающей и пишущей
func (c *Connection) Run(ctx context.Context) {
	go c.writePump(ctx)
	go c.readPump(ctx)
}

// читаем сообщения от клиента
func (c *Connection) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		c.Close()
	}()

	c.conn.SetReadLimit(1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		// пока просто парсим и игнорим — скелет
		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// TODO: дальше сюда воткнём прокидку в MatchService
	}
}

// пишем сообщения клиенту
func (c *Connection) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Helper: отправить тестовый state
func (c *Connection) SendTestWelcome() {
	data, _ := json.Marshal(ServerMessage{
		Type: "hello",
		Data: map[string]any{
			"battle_id": c.battleID.String(),
			"user_id":   c.userID.String(),
			"ts":        time.Now().Unix(),
		},
	})
	c.send <- data
}
