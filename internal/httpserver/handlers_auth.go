package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sundabaoh-rgb/tankionline/internal/auth"
)

type AuthHandlers struct {
	authService auth.Service
}

func NewAuthHandlers(a auth.Service) *AuthHandlers {
	return &AuthHandlers{authService: a}
}

func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nickname string `json:"nickname"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	tokens, err := h.authService.Register(r.Context(), req.Nickname, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    tokens,
	})
}

func (h *AuthHandlers) Me(w http.ResponseWriter, r *http.Request) {
	raw := GetUserID(r.Context())
	id, ok := raw.(uuid.UUID)
	if !ok {
		http.Error(w, "invalid user id in context", http.StatusInternalServerError)
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), id)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	dto := auth.ToUserDTO(user)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    dto,
	})
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Nickname string `json:"nickname"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	tokens, err := h.authService.Login(r.Context(), req.Nickname, req.Password)
	if err != nil {
		// для безопасности не палим, что именно не так
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    tokens,
	})
}

func (h *AuthHandlers) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	tokens, err := h.authService.RefreshTokens(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    tokens,
	})
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	accessToken := strings.TrimPrefix(header, "Bearer ")

	if err := h.authService.Logout(r.Context(), accessToken); err != nil {
		http.Error(w, "failed to logout", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    "ok",
	})
}
