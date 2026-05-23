package room

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
	"github.com/sundabaoh-rgb/tankionline/internal/logger"
)

var (
	ErrUserAlreadyInRoom = errors.New("user already in another room")
	ErrUserNotInRoom     = errors.New("user not in this room")
	ErrRoomFull          = errors.New("room is full")
)

type Repository interface {
	Create(ctx context.Context, r *domain.Room) error
	List(ctx context.Context) ([]*domain.Room, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) // !!!
}

type PlayerRepository interface {
	Join(ctx context.Context, roomID, userID uuid.UUID) error
	Leave(ctx context.Context, roomID, userID uuid.UUID) error
	IsInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	GetUserRoom(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error)
	ListPlayers(ctx context.Context, roomID uuid.UUID) ([]PlayerInRoom, error)
	IsInAnyRoom(ctx context.Context, userID uuid.UUID) (bool, error)
	CountInRoom(ctx context.Context, roomID uuid.UUID) (int, error)
}

type Service interface {
	CreateRoom(ctx context.Context, ownerID uuid.UUID, in CreateRoomInput) (*domain.Room, error)
	ListRooms(ctx context.Context) ([]*domain.Room, error)
	JoinRoom(ctx context.Context, roomID, userID uuid.UUID) error
	LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error
	ListRoomPlayers(ctx context.Context, roomID uuid.UUID) ([]PlayerInRoom, error)
}

type CreateRoomInput struct {
	Name            string  `json:"name"`
	MaxPlayers      int     `json:"max_players"`
	DurationMinutes int     `json:"duration_minutes"`
	GoldPerKill     int     `json:"gold_per_kill"`
	FundModifier    float64 `json:"fund_modifier"`
}

type service struct {
	repo    Repository
	players PlayerRepository
	logger  logger.Logger
}

type PlayerInRoom struct {
	ID       uuid.UUID `json:"id"`
	Nickname string    `json:"nickname"`
}

func NewService(repo Repository, players PlayerRepository, log logger.Logger) Service {
	return &service{
		repo:    repo,
		players: players,
		logger:  log.Named("room_service"),
	}
}

func (s *service) CreateRoom(ctx context.Context, ownerID uuid.UUID, in CreateRoomInput) (*domain.Room, error) {
	room, err := domain.NewRoom(
		ownerID,
		in.Name,
		in.MaxPlayers,
		in.DurationMinutes,
		in.GoldPerKill,
		in.FundModifier,
	)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

func (s *service) ListRooms(ctx context.Context) ([]*domain.Room, error) {
	return s.repo.List(ctx)
}

func (s *service) ListRoomPlayers(ctx context.Context, roomID uuid.UUID) ([]PlayerInRoom, error) {
	return s.players.ListPlayers(ctx, roomID)
}

func (s *service) JoinRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	inRoom, err := s.players.IsInRoom(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if inRoom {
		return nil
	}

	currentRoom, err := s.players.GetUserRoom(ctx, userID)
	if err != nil {
		return err
	}
	if currentRoom != nil {
		if *currentRoom == roomID {
			return nil
		}
		return errors.New("user is already in another room")
	}

	rm, err := s.repo.GetByID(ctx, roomID)
	if err != nil {
		return err
	}

	count, err := s.players.CountInRoom(ctx, roomID)
	if err != nil {
		return err
	}

	if count >= rm.MaxPlayers() {
		return ErrRoomFull
	}

	return s.players.Join(ctx, roomID, userID)
}

func (s *service) LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	inRoom, err := s.players.IsInRoom(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if !inRoom {
		return nil
	}
	return s.players.Leave(ctx, roomID, userID)
}
