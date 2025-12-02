package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/sundabaoh-rgb/tankionline/internal/httpserver/response"
	"github.com/sundabaoh-rgb/tankionline/internal/match"
)

type DebugHandler struct {
	match *match.Service
}

func NewDebugHandler(m *match.Service) *DebugHandler {
	return &DebugHandler{match: m}
}

// POST /api/v1/debug/battles/start?room_id=...
func (h *DebugHandler) StartBattle(w http.ResponseWriter, r *http.Request) {
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

	// временно жёстко: 15 минут
	m, err := h.match.EnsureBattleRunning(r.Context(), roomID, 15)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	b := m.Battle

	response.OK(w, map[string]any{
		"battle_id":   b.ID().String(),
		"room_id":     b.RoomID().String(),
		"status":      b.Status(),
		"started_at":  b.StartedAt(),
		"ends_at":     b.EndsAt(),
		"finished_at": b.FinishedAt(),
		"fund":        b.Fund(),
	})
}
