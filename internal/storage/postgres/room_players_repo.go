package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
	"github.com/sundabaoh-rgb/tankionline/internal/room"
)

type RoomPlayersRepo struct {
	pool *pgxpool.Pool
}

func NewRoomPlayersRepo(pool *pgxpool.Pool) *RoomPlayersRepo {
	return &RoomPlayersRepo{pool: pool}
}

func (r *RoomPlayersRepo) Join(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO room_players (room_id, user_id)
         VALUES ($1, $2)
         ON CONFLICT DO NOTHING`,
		roomID, userID,
	)
	return err
}

func (r *RoomPlayersRepo) Leave(ctx context.Context, roomID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM room_players
         WHERE room_id = $1 AND user_id = $2`,
		roomID, userID,
	)
	return err
}

func (r *RoomPlayersRepo) IsInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
            SELECT 1
            FROM room_players
            WHERE room_id = $1 AND user_id = $2
        )`,
		roomID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *RoomPlayersRepo) GetUserRoom(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	var roomID uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT room_id FROM room_players WHERE user_id = $1 LIMIT 1`,
		userID,
	).Scan(&roomID)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &roomID, nil
}

func (r *RoomPlayersRepo) ListPlayers(ctx context.Context, roomID uuid.UUID) ([]room.PlayerInRoom, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id, u.nickname
         FROM room_players rp
         JOIN users u ON u.id = rp.user_id
         WHERE rp.room_id = $1
         ORDER BY u.nickname`,
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []room.PlayerInRoom
	for rows.Next() {
		var p room.PlayerInRoom
		if err := rows.Scan(&p.ID, &p.Nickname); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *RoomPlayersRepo) IsInAnyRoom(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM room_players WHERE user_id = $1)`,
		userID,
	).Scan(&exists)
	return exists, err
}

func (r *RoomPlayersRepo) CountInRoom(ctx context.Context, roomID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM room_players WHERE room_id = $1`,
		roomID,
	).Scan(&count)
	return count, err
}

func (r *RoomRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	var (
		roomID       uuid.UUID
		name         string
		ownerID      uuid.UUID
		maxPlayers   int
		durationMin  int
		goldPerKill  int
		fundModifier float64
		createdAt    time.Time
	)

	err := r.pool.QueryRow(ctx,
		`SELECT id, name, owner_id, max_players, duration_minutes,
                gold_per_kill, fund_modifier, created_at
         FROM rooms
         WHERE id = $1`,
		id,
	).Scan(&roomID, &name, &ownerID, &maxPlayers, &durationMin,
		&goldPerKill, &fundModifier, &createdAt)

	if err != nil {
		return nil, err
	}

	room := domain.NewRoomFromDB(
		roomID,
		name,
		ownerID,
		maxPlayers,
		durationMin,
		goldPerKill,
		fundModifier,
		createdAt,
	)

	return room, nil
}
