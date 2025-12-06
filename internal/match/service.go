package match

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/sundabaoh-rgb/tankionline/internal/battle"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
	"github.com/sundabaoh-rgb/tankionline/internal/logger"
	"github.com/sundabaoh-rgb/tankionline/internal/ws"
)

type Service struct {
	log logger.Logger

	battles battle.Repository

	hub *ws.Hub

	mu      sync.Mutex
	matches map[uuid.UUID]*Match
}

func NewService(battleRepo battle.Repository, hub *ws.Hub, log logger.Logger) *Service {
	return &Service{
		log:     log.Named("match_service"),
		battles: battleRepo,
		hub:     hub,
		matches: make(map[uuid.UUID]*Match),
	}
}

// раньше у тебя уже была EnsureBattleRunning(ctx, roomID, durationMinutes)
// просто чуть допилим, чтобы она создавала Match и стартовала Run
func (s *Service) EnsureBattleRunning(ctx context.Context, roomID uuid.UUID, durationMinutes int) (*Match, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. ищем активную битву в репо
	b, err := s.battles.GetActiveByRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	// если матчи уже есть — возвр
	if m, ok := s.matches[b.ID()]; ok {
		return m, nil
	}

	// 2. если матча нет — создаём
	m := NewMatch(b.ID(), s.hub, s.log)
	s.matches[b.ID()] = m

	// запускаем цикл в отдельной горутине
	go m.Run(ctx)

	return m, nil
}

func (s *Service) AddPlayer(ctx context.Context, battleID, userID uuid.UUID) {
	s.mu.Lock()
	m := s.matches[battleID]
	s.mu.Unlock()
	if m == nil {
		return
	}
	m.AddPlayer(userID)
}

func (s *Service) RemovePlayer(battleID, userID uuid.UUID) {
	s.mu.Lock()
	m := s.matches[battleID]
	s.mu.Unlock()
	if m == nil {
		return
	}
	m.RemovePlayer(userID)
}

func (s *Service) HandleInput(battleID, userID uuid.UUID, in domain.PlayerInput) {
	s.mu.Lock()
	m := s.matches[battleID]
	s.mu.Unlock()
	if m == nil {
		return
	}
	m.SetInput(userID, in)
}
