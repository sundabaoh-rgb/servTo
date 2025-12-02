package battle

import (
	"context"

	"github.com/google/uuid"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
)

type Repository interface {
	Create(ctx context.Context, b *domain.Battle) error
	Update(ctx context.Context, b *domain.Battle) error
	GetActiveByRoom(ctx context.Context, roomID uuid.UUID) (*domain.Battle, error)
}

type StatsRepository interface {
	InsertMany(ctx context.Context, stats []domain.PlayerBattleStats) error
}
