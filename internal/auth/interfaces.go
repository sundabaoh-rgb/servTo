package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByNickname(ctx context.Context, nickname string) (*domain.User, error)
}

type TokenManager interface {
	NewTokens(ctx context.Context, userID uuid.UUID) (*Tokens, error)
	Refresh(ctx context.Context, refreshToken string) (*Tokens, error)
	ValidateAccess(token string) (uuid.UUID, error)
	Logout(ctx context.Context, accessToken string) error
}

type Service interface {
	Register(ctx context.Context, nickname, password string) (*Tokens, error)
	Login(ctx context.Context, nickname, password string) (*Tokens, error)
	Logout(ctx context.Context, accessToken string) error

	RefreshTokens(ctx context.Context, refreshToken string) (*Tokens, error)
	ValidateAccessToken(token string) (uuid.UUID, error)

	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}
