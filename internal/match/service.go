package match

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/sundabaoh-rgb/tankionline/internal/battle"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
	"github.com/sundabaoh-rgb/tankionline/internal/logger"
)

type Match struct {
	Battle *domain.Battle
	// позже: состояние игроков, пуль и т.п.
}

type Service struct {
	mu      sync.RWMutex
	matches map[uuid.UUID]*Match // battleID -> match

	battles battle.Repository
	stats   battle.StatsRepository
	log     logger.Logger
}

func NewService(
	battles battle.Repository,
	stats battle.StatsRepository,
	log logger.Logger,
) *Service {
	return &Service{
		matches: make(map[uuid.UUID]*Match),
		battles: battles,
		stats:   stats,
		log:     log.Named("match_service"),
	}
}

// EnsureBattleRunning либо возвращает уже запущенный матч,
// либо создаёт новый battle + match.
func (s *Service) EnsureBattleRunning(ctx context.Context, roomID uuid.UUID, roomDurationMinutes int) (*Match, error) {
	// 1. ищем активную битву по комнате
	b, err := s.battles.GetActiveByRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if b == nil {
		// 2. создаём новую битву
		b, err = domain.NewBattle(roomID, roomDurationMinutes)
		if err != nil {
			return nil, err
		}
		if err := s.battles.Create(ctx, b); err != nil {
			return nil, err
		}
	}

	// 3. достаём/создаём Match в памяти
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.matches[b.ID()]
	if ok {
		return m, nil
	}

	m = &Match{Battle: b}
	s.matches[b.ID()] = m

	return m, nil
}
