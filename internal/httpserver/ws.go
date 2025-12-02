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
	// для локалки: разрешаем любые origin'ы
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleGameWS(w http.ResponseWriter, r *http.Request) {
	// 1. достаём токен и room_id из query
	token := r.URL.Query().Get("token")
	if token == "" {
		response.Unauthorized(w, "missing token")
		return
	}

	roomIDStr := r.URL.Query().Get("room_id")
	if roomIDStr == "" {
		response.BadRequest(w, "missing room_id")
		return
	}

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		response.BadRequest(w, "invalid room_id")
		return
	}

	// 2. валидируем токен -> userID
	userID, err := s.authService.ValidateAccessToken(token)
	if err != nil {
		response.Unauthorized(w, "invalid token")
		return
	}

	// 3. пока не тянем комнату из БД — просто 15 минут
	match, err := s.matchService.EnsureBattleRunning(r.Context(), roomID, 15)
	if err != nil || match == nil {
		response.InternalError(w, "failed to start battle")
		return
	}
	battleID := match.Battle.ID()

	// 4. апгрейд в WebSocket
	wsConn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// 5. создаём Connection и регистрируем в хабе
	ctx, cancel := context.WithCancel(r.Context())

	c := ws.NewConnection(s.wsHub, wsConn, battleID, userID)
	s.wsHub.Register(c) // небольшой хелпер, см. ниже
	c.Run(ctx)
	c.SendTestWelcome()

	// контекст закроется, когда соединение упадёт
	go func() {
		<-ctx.Done()
		cancel()
	}()
}
