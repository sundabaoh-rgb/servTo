package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
	"github.com/sundabaoh-rgb/tankionline/internal/logger"
)

type service struct {
	users  UserRepository
	tokens TokenManager
	hasher PasswordHasher
	logger logger.Logger
}

func NewService(
	users UserRepository,
	tokens TokenManager,
	hasher PasswordHasher,
	log logger.Logger,
) Service {
	return &service{
		users:  users,
		tokens: tokens,
		hasher: hasher,
		logger: log,
	}
}

func (s *service) Register(ctx context.Context, nickname, password string) (*Tokens, error) {
	if nickname == "" || password == "" {
		return nil, ErrInvalidInput
	}

	// проверяем занят ли ник
	_, err := s.users.GetByNickname(ctx, nickname)
	if err == nil {
		return nil, ErrNicknameTaken
	}

	// создаём hashedPassword
	hashed, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	// создаём ID заранее
	userID := uuid.New()

	// создаём доменного юзера
	user, err := domain.NewUser(userID, nickname, hashed)
	if err != nil {
		return nil, err
	}

	// сохраняем в БД
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	// токены
	tokens, err := s.tokens.NewTokens(ctx, user.ID())
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *service) Login(ctx context.Context, nickname, password string) (*Tokens, error) {
	if nickname == "" || password == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.users.GetByNickname(ctx, nickname)
	if err != nil {
		s.logger.Warn("login: user not found", "nickname", nickname, "err", err)
		return nil, ErrInvalidLogin
	}

	if !user.VerifyPassword(password) {
		s.logger.Warn("login: invalid password", "nickname", nickname)
		return nil, ErrInvalidLogin
	}

	tokens, err := s.tokens.NewTokens(ctx, user.ID())
	if err != nil {
		s.logger.Error("login: failed to create tokens", "err", err)
		return nil, err
	}

	return tokens, nil
}

func (s *service) RefreshTokens(ctx context.Context, refreshToken string) (*Tokens, error) {
	return s.tokens.Refresh(ctx, refreshToken)
}

func (s *service) ValidateAccessToken(token string) (uuid.UUID, error) {
	return s.tokens.ValidateAccess(token)
}

func (s *service) Logout(ctx context.Context, accessToken string) error {
	return s.tokens.Logout(ctx, accessToken)
}

func (s *service) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}
