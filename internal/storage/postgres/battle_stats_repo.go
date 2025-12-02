package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sundabaoh-rgb/tankionline/internal/battle"
	"github.com/sundabaoh-rgb/tankionline/internal/domain"
)

var _ battle.StatsRepository = (*BattleStatsRepo)(nil)

type BattleStatsRepo struct {
	pool *pgxpool.Pool
}

func NewBattleStatsRepo(pool *pgxpool.Pool) *BattleStatsRepo {
	return &BattleStatsRepo{pool: pool}
}

func (r *BattleStatsRepo) InsertMany(ctx context.Context, stats []domain.PlayerBattleStats) error {
	if len(stats) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, s := range stats {
		batch.Queue(`
			INSERT INTO battle_players (
				battle_id, user_id, kills, deaths, damage_dealt,
				damage_taken, score, reward, xp
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`,
			s.BattleID,
			s.UserID,
			s.Kills,
			s.Deaths,
			s.DamageDealt,
			s.DamageTaken,
			s.Score,
			s.Reward,
			s.XP,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	// прогоняем результаты, чтобы поймать ошибки
	for range stats {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}
