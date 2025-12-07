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

	// 1. Пытаемся найти активную битву в БД
	b, err := s.battles.GetActiveByRoom(ctx, roomID)
	if err != nil {
		// реальная ошибка (БД сдохла и т.п.) — валимся
		return nil, err
	}

	// 1.1. Активной битвы нет — создаём новую
	if b == nil {
		s.log.Info("match_service: no active battle for room, creating new",
			"room_id", roomID,
		)

		// создаём доменную битву
		b, err = domain.NewBattle(roomID, durationMinutes)
		if err != nil {
			return nil, err
		}

		// сохраняем в репозиторий
		if err := s.battles.Create(ctx, b); err != nil {
			return nil, err
		}
	}

	battleID := b.ID()

	// 2. Если матч уже есть в памяти — просто возвращаем
	if m, ok := s.matches[battleID]; ok {
		return m, nil
	}

	// 3. Нет матча — создаём
	m := NewMatch(battleID, s.hub, s.log)
	s.matches[battleID] = m

	// 4. Запускаем игровой цикл
	// 4. Запускаем игровой цикл на фоне, независимо от r.Context()
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
