package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/sundabaoh-rgb/tankionline/internal/domain"
	"github.com/sundabaoh-rgb/tankionline/internal/room"
)

type RoomHandlers struct {
	service room.Service
}

func NewRoomHandlers(s room.Service) *RoomHandlers {
	return &RoomHandlers{service: s}
}

// DTO для ответа
type RoomDTO struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	OwnerID         string  `json:"owner_id"`
	MaxPlayers      int     `json:"max_players"`
	DurationMinutes int     `json:"duration_minutes"`
	GoldPerKill     int     `json:"gold_per_kill"`
	FundModifier    float64 `json:"fund_modifier"`
	CreatedAt       string  `json:"created_at"`
}

func toRoomDTO(r *domain.Room) RoomDTO {
	return RoomDTO{
		ID:              r.ID().String(),
		Name:            r.Name(),
		OwnerID:         r.OwnerID().String(),
		MaxPlayers:      r.MaxPlayers(),
		DurationMinutes: r.DurationMinutes(),
		GoldPerKill:     r.GoldPerKill(),
		FundModifier:    r.FundModifier(),
		CreatedAt:       r.CreatedAt().Format(time.RFC3339),
	}
}

func (h *RoomHandlers) List(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.service.ListRooms(r.Context())
	if err != nil {
		http.Error(w, "failed to list rooms", http.StatusInternalServerError)
		return
	}

	out := make([]RoomDTO, 0, len(rooms))
	for _, rm := range rooms {
		out = append(out, toRoomDTO(rm))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    out,
	})
}

func (h *RoomHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var req room.CreateRoomInput

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	rawID := GetUserID(r.Context())
	ownerID, ok := rawID.(uuid.UUID)
	if !ok {
		http.Error(w, "invalid user id", http.StatusInternalServerError)
		return
	}

	rm, err := h.service.CreateRoom(r.Context(), ownerID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dto := toRoomDTO(rm)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    dto,
	})
}

func (h *RoomHandlers) Join(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "roomID")
	roomID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	raw := GetUserID(r.Context())
	userID, ok := raw.(uuid.UUID)
	if !ok {
		http.Error(w, "invalid user id in context", http.StatusUnauthorized)
		return
	}

	if err := h.service.JoinRoom(r.Context(), roomID, userID); err != nil {
		if errors.Is(err, room.ErrRoomFull) {
			http.Error(w, "room is full", http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to join room", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
	})
}

func (h *RoomHandlers) Leave(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "roomID")
	roomID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	raw := GetUserID(r.Context())
	userID, ok := raw.(uuid.UUID)
	if !ok {
		http.Error(w, "invalid user id in context", http.StatusUnauthorized)
		return
	}

	if err := h.service.LeaveRoom(r.Context(), roomID, userID); err != nil {
		if errors.Is(err, room.ErrUserNotInRoom) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to leave room", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
	})
}

func (h *RoomHandlers) Players(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "roomID")
	roomID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	players, err := h.service.ListRoomPlayers(r.Context(), roomID)
	if err != nil {
		http.Error(w, "failed to list players", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    players,
	})
}
