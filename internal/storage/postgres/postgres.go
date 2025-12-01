package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
)

const defaultConnectTimeout = 5 * time.Second

// -----------------------------------------------------------------------------
// POOL INIT
// -----------------------------------------------------------------------------
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("empty Postgres DSN")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db failed: %w", err)
	}

	return pool, nil
}

// -----------------------------------------------------------------------------
// USER REPO
// -----------------------------------------------------------------------------
type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, nickname, role, xp, created_at, password_hash)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		u.ID(),
		u.Nickname(),
		u.Role().String(),
		u.XP(),
		u.CreatedAt(),
		u.PasswordHash(),
	)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var (
		uid          uuid.UUID
		nickname     string
		roleStr      string
		xp           int
		createdAt    time.Time
		passwordHash string
	)

	err := r.pool.QueryRow(ctx,
		`SELECT id, nickname, role, xp, created_at, password_hash
		 FROM users
		 WHERE id = $1`,
		id,
	).Scan(&uid, &nickname, &roleStr, &xp, &createdAt, &passwordHash)

	if err != nil {
		return nil, err
	}

	user, err := domain.NewUserFromDB(uid, nickname, domain.UserRole(roleStr), xp, createdAt, passwordHash)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepo) GetByNickname(ctx context.Context, nickname string) (*domain.User, error) {
	var (
		uid          uuid.UUID
		nick         string
		roleStr      string
		xp           int
		createdAt    time.Time
		passwordHash string
	)

	err := r.pool.QueryRow(ctx,
		`SELECT id, nickname, role, xp, created_at, password_hash
		 FROM users
		 WHERE nickname = $1`,
		nickname,
	).Scan(&uid, &nick, &roleStr, &xp, &createdAt, &passwordHash)

	if err != nil {
		return nil, err
	}

	user, err := domain.NewUserFromDB(uid, nick, domain.UserRole(roleStr), xp, createdAt, passwordHash)
	if err != nil {
		return nil, err
	}

	return user, nil
}
