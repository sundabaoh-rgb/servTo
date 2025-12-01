package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
)

type RoomRepo struct {
	pool *pgxpool.Pool
}

func NewRoomRepo(pool *pgxpool.Pool) *RoomRepo {
	return &RoomRepo{pool: pool}
}

func (r *RoomRepo) Create(ctx context.Context, room *domain.Room) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO rooms (id, name, owner_id, max_players, duration_minutes, gold_per_kill, fund_modifier, created_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		room.ID(),
		room.Name(),
		room.OwnerID(),
		room.MaxPlayers(),
		room.DurationMinutes(),
		room.GoldPerKill(),
		room.FundModifier(),
		room.CreatedAt(),
	)
	return err
}

func (r *RoomRepo) List(ctx context.Context) ([]*domain.Room, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, owner_id, max_players, duration_minutes, gold_per_kill, fund_modifier, created_at
         FROM rooms
         ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*domain.Room

	for rows.Next() {
		var (
			id              uuid.UUID
			name            string
			ownerID         uuid.UUID
			maxPlayers      int
			durationMinutes int
			goldPerKill     int
			fundModifier    float64
			createdAt       time.Time
		)

		if err := rows.Scan(&id, &name, &ownerID, &maxPlayers, &durationMinutes, &goldPerKill, &fundModifier, &createdAt); err != nil {
			return nil, err
		}

		rooms = append(rooms, domain.NewRoomFromDB(
			id, name, ownerID, maxPlayers, durationMinutes, goldPerKill, fundModifier, createdAt,
		))
	}

	return rooms, rows.Err()
}
