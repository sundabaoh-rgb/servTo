package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/sundabaoh-rgb/tankionline/internal/domain"
	"github.com/sundabaoh-rgb/tankionline/internal/httpserver/api"
	dto "github.com/sundabaoh-rgb/tankionline/internal/httpserver/api"
	"github.com/sundabaoh-rgb/tankionline/internal/httpserver/middleware"
	"github.com/sundabaoh-rgb/tankionline/internal/httpserver/response"
	"github.com/sundabaoh-rgb/tankionline/internal/room"
)

type RoomHandler struct {
	*BaseHandler
	service room.Service
}

func NewRoomHandler(service room.Service) *RoomHandler {
	return &RoomHandler{
		BaseHandler: &BaseHandler{},
		service:     service,
	}
}

func (h *RoomHandler) List(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.service.ListRooms(r.Context())
	if err != nil {
		response.InternalError(w, "failed to list rooms")
		return
	}

	out := make([]dto.RoomResponse, 0, len(rooms))
	for _, rm := range rooms {
		out = append(out, h.toRoomDTO(rm))
	}

	response.OK(w, out)
}

func (h *RoomHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateRoomRequest
	if !h.BindJSON(w, r, &req) {
		return
	}

	userID := middleware.MustGetUserID(r.Context())

	// Конвертируем DTO во внутренний формат
	input := room.CreateRoomInput{
		Name:            req.Name,
		MaxPlayers:      req.MaxPlayers,
		DurationMinutes: req.DurationMinutes,
		GoldPerKill:     req.GoldPerKill,
		FundModifier:    req.FundModifier,
	}

	rm, err := h.service.CreateRoom(r.Context(), userID, input)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	response.Created(w, h.toRoomDTO(rm))
}

func (h *RoomHandler) Join(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "roomID")
	roomID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid room id")
		return
	}

	userID := middleware.MustGetUserID(r.Context())

	if err := h.service.JoinRoom(r.Context(), roomID, userID); err != nil {
		if errors.Is(err, room.ErrRoomFull) {
			response.BadRequest(w, "room is full")
			return
		}
		if errors.Is(err, room.ErrUserAlreadyInRoom) {
			response.BadRequest(w, "user already in this room")
			return
		}
		response.InternalError(w, "failed to join room")
		return
	}

	response.OK(w, "joined successfully")
}

func (h *RoomHandler) Leave(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "roomID")
	roomID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid room id")
		return
	}

	userID := middleware.MustGetUserID(r.Context())

	if err := h.service.LeaveRoom(r.Context(), roomID, userID); err != nil {
		if errors.Is(err, room.ErrUserNotInRoom) {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalError(w, "failed to leave room")
		return
	}

	response.OK(w, "left successfully")
}

func (h *RoomHandler) Players(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "roomID")
	roomID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, "invalid room id")
		return
	}

	players, err := h.service.ListRoomPlayers(r.Context(), roomID)
	if err != nil {
		response.InternalError(w, "failed to list players")
		return
	}

	out := make([]api.RoomPlayer, 0, len(players))
	for _, u := range players {
		out = append(out, api.RoomPlayer{
			ID:       u.ID.String(),
			Nickname: u.Nickname,
			Ready:    false, // пока заглушка, потом можно добавить флаг готовности
		})
	}

	response.OK(w, api.RoomPlayersResponse{
		Players: out,
	})
}

func (h *RoomHandler) toRoomDTO(r *domain.Room) dto.RoomResponse {
	return dto.RoomResponse{
		ID:              r.ID().String(),
		Name:            r.Name(),
		OwnerID:         r.OwnerID().String(),
		MaxPlayers:      r.MaxPlayers(),
		DurationMinutes: r.DurationMinutes(),
		GoldPerKill:     r.GoldPerKill(),
		FundModifier:    r.FundModifier(),
		CreatedAt:       r.CreatedAt(),
	}
}
