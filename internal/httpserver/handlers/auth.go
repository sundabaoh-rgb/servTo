package handlers

import (
	"net/http"
	"strings"

	"github.com/sundabaoh-rgb/tankionline/internal/auth"
	dto "github.com/sundabaoh-rgb/tankionline/internal/httpserver/api"
	"github.com/sundabaoh-rgb/tankionline/internal/httpserver/middleware"
	"github.com/sundabaoh-rgb/tankionline/internal/httpserver/response"
)

type AuthHandler struct {
	*BaseHandler
	service auth.Service
}

func NewAuthHandler(service auth.Service) *AuthHandler {
	return &AuthHandler{
		BaseHandler: &BaseHandler{},
		service:     service,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if !h.BindJSON(w, r, &req) {
		return
	}

	tokens, err := h.service.Register(r.Context(), req.Nickname, req.Password)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	response.Created(w, tokens)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		response.Unauthorized(w, "authentication required")
		return
	}

	user, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		response.NotFound(w, "user not found")
		return
	}

	dto := auth.ToUserDTO(user)
	response.OK(w, dto)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req dto.LoginRequest
	if !h.BindJSON(w, r, &req) {
		return
	}

	tokens, err := h.service.Login(r.Context(), req.Nickname, req.Password)
	if err != nil {
		response.Unauthorized(w, "invalid credentials")
		return
	}

	response.OK(w, tokens)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if !h.RequireMethod(w, r, http.MethodPost) {
		return
	}

	var req dto.RefreshRequest
	if !h.BindJSON(w, r, &req) {
		return
	}

	tokens, err := h.service.RefreshTokens(r.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(w, "invalid refresh token")
		return
	}

	response.OK(w, tokens)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if !h.RequireMethod(w, r, http.MethodPost) {
		return
	}

	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		response.Unauthorized(w, "missing token")
		return
	}
	accessToken := strings.TrimPrefix(header, "Bearer ")

	if err := h.service.Logout(r.Context(), accessToken); err != nil {
		response.InternalError(w, "failed to logout")
		return
	}

	response.OK(w, "ok")
}
