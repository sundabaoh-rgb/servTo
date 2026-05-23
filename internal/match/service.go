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

func (s *Service) EnsureBattleRunning(ctx context.Context, roomID uuid.UUID, durationMinutes int) (*Match, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := s.battles.GetActiveByRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if b == nil {
		s.log.Info("match_service: no active battle for room, creating new",
			"room_id", roomID,
		)

		b, err = domain.NewBattle(roomID, durationMinutes)
		if err != nil {
			return nil, err
		}

		if err := s.battles.Create(ctx, b); err != nil {
			return nil, err
		}
	}

	battleID := b.ID()

	if m, ok := s.matches[battleID]; ok {
		return m, nil
	}

	m := NewMatch(battleID, s.hub, s.log)
	s.matches[battleID] = m

	go m.Run(context.Background())

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
