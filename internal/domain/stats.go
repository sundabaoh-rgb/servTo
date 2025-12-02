package domain

import "github.com/google/uuid"

type PlayerBattleStats struct {
	BattleID uuid.UUID
	UserID   uuid.UUID

	Kills       int
	Deaths      int
	DamageDealt int
	DamageTaken int
	Score       int
	Reward      int64
	XP          int
}
