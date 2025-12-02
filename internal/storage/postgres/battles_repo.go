package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sundabaoh-rgb/tankionline/internal/battle"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
)

var _ battle.Repository = (*BattleRepo)(nil)

type BattleRepo struct {
	pool *pgxpool.Pool
}

func NewBattleRepo(pool *pgxpool.Pool) *BattleRepo {
	return &BattleRepo{pool: pool}
}

func (r *BattleRepo) Create(ctx context.Context, b *domain.Battle) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO battles (id, room_id, started_at, ends_at, finished_at, status, fund)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		b.ID(),
		b.RoomID(),
		b.StartedAt(),
		b.EndsAt(),
		b.FinishedAt(),
		b.Status(),
		b.Fund(),
	)
	return err
}

func (r *BattleRepo) Update(ctx context.Context, b *domain.Battle) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE battles
		SET finished_at = $2,
		    status      = $3,
		    fund        = $4
		WHERE id = $1
	`,
		b.ID(),
		b.FinishedAt(),
		b.Status(),
		b.Fund(),
	)
	return err
}

func (r *BattleRepo) GetActiveByRoom(ctx context.Context, roomID uuid.UUID) (*domain.Battle, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, room_id, started_at, ends_at, finished_at, status, fund
		FROM battles
		WHERE room_id = $1 AND status = 'active'
		LIMIT 1
	`, roomID)

	var (
		id         uuid.UUID
		dbRoomID   uuid.UUID
		startedAt  time.Time
		endsAt     time.Time
		finishedAt *time.Time
		status     domain.BattleStatus
		fund       int64
	)

	err := row.Scan(&id, &dbRoomID, &startedAt, &endsAt, &finishedAt, &status, &fund)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return domain.NewBattleFromDB(id, dbRoomID, startedAt, endsAt, finishedAt, status, fund), nil
}
