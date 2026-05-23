package httpserver

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/sundabaoh-rgb/tankionline/internal/httpserver/response"
	"github.com/sundabaoh-rgb/tankionline/internal/ws"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (s *Server) handleGameWS(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	roomIDStr := q.Get("room_id")
	token := q.Get("token")

	s.log.Info("ws.game: incoming",
		"room_id", roomIDStr,
		"token_prefix", func() string {
			if len(token) < 8 {
				return token
			}
			return token[:8]
		}(),
	)

	if token == "" {
		s.log.Warn("ws.game: missing token")
		response.Unauthorized(w, "missing token")
		return
	}

	if roomIDStr == "" {
		s.log.Warn("ws.game: missing room_id")
		response.BadRequest(w, "missing room_id")
		return
	}

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		s.log.Warn("ws.game: invalid room_id", "err", err)
		response.BadRequest(w, "invalid room_id")
		return
	}

	userID, err := s.authService.ValidateAccessToken(token)
	if err != nil {
		s.log.Warn("ws.game: invalid token", "err", err)
		response.Unauthorized(w, "invalid token")
		return
	}

	m, err := s.matchService.EnsureBattleRunning(r.Context(), roomID, 15)
	if err != nil || m == nil {
		s.log.Error("ws.game: EnsureBattleRunning failed", "err", err)
		response.InternalError(w, "failed to start battle")
		return
	}

	battleID := m.BattleID
	s.log.Info("ws.game: battle running", "battle_id", battleID.String())

	wsConn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("ws.game: ws upgrade failed", "err", err)
		return
	}

	ctx := context.Background()

	s.matchService.AddPlayer(ctx, battleID, userID)

	c := ws.NewConnection(
		s.wsHub,
		wsConn,
		battleID,
		userID,
		s.matchService.HandleInput,
		s.matchService.RemovePlayer,
	)

	s.wsHub.Register(c)
	c.Run(ctx)
	c.SendTestWelcome()
}
