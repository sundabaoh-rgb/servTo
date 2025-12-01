package auth

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type memoryTokenManager struct {
	mu      sync.RWMutex
	access  map[string]uuid.UUID
	refresh map[string]uuid.UUID
}

func NewMemoryTokenManager() TokenManager {
	return &memoryTokenManager{
		access:  make(map[string]uuid.UUID),
		refresh: make(map[string]uuid.UUID),
	}
}

func (m *memoryTokenManager) NewTokens(ctx context.Context, userID uuid.UUID) (*Tokens, error) {
	access := uuid.NewString()
	refresh := uuid.NewString()

	m.mu.Lock()
	m.access[access] = userID
	m.refresh[refresh] = userID
	m.mu.Unlock()

	return &Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func (m *memoryTokenManager) Refresh(ctx context.Context, refreshToken string) (*Tokens, error) {
	m.mu.RLock()
	userID, ok := m.refresh[refreshToken]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrInvalidToken
	}

	newAccess := uuid.NewString()

	m.mu.Lock()
	m.access[newAccess] = userID
	m.mu.Unlock()

	return &Tokens{AccessToken: newAccess, RefreshToken: refreshToken}, nil
}

func (m *memoryTokenManager) ValidateAccess(token string) (uuid.UUID, error) {
	m.mu.RLock()
	userID, ok := m.access[token]
	m.mu.RUnlock()

	if !ok {
		return uuid.Nil, ErrInvalidToken
	}

	return userID, nil
}

func (m *memoryTokenManager) Logout(ctx context.Context, accessToken string) error {
	m.mu.Lock()
	delete(m.access, accessToken)
	m.mu.Unlock()
	return nil
}
