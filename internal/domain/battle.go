package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type BattleStatus string

const (
	BattleStatusActive    BattleStatus = "active"
	BattleStatusFinished  BattleStatus = "finished"
	BattleStatusCancelled BattleStatus = "cancelled"
)

type Battle struct {
	id         uuid.UUID
	roomID     uuid.UUID
	startedAt  time.Time
	endsAt     time.Time
	finishedAt *time.Time
	status     BattleStatus
	fund       int64
}

func NewBattle(roomID uuid.UUID, durationMinutes int) (*Battle, error) {
	if durationMinutes <= 0 {
		return nil, errors.New("durationMinutes must be > 0")
	}
	now := time.Now().UTC()
	endsAt := now.Add(time.Duration(durationMinutes) * time.Minute)

	return &Battle{
		id:        uuid.New(),
		roomID:    roomID,
		startedAt: now,
		endsAt:    endsAt,
		status:    BattleStatusActive,
		fund:      0,
	}, nil
}

// Для чтения из БД
func NewBattleFromDB(
	id, roomID uuid.UUID,
	startedAt, endsAt time.Time,
	finishedAt *time.Time,
	status BattleStatus,
	fund int64,
) *Battle {
	return &Battle{
		id:         id,
		roomID:     roomID,
		startedAt:  startedAt,
		endsAt:     endsAt,
		finishedAt: finishedAt,
		status:     status,
		fund:       fund,
	}
}

// --- геттеры ---

func (b *Battle) ID() uuid.UUID          { return b.id }
func (b *Battle) RoomID() uuid.UUID      { return b.roomID }
func (b *Battle) StartedAt() time.Time   { return b.startedAt }
func (b *Battle) EndsAt() time.Time      { return b.endsAt }
func (b *Battle) FinishedAt() *time.Time { return b.finishedAt }
func (b *Battle) Status() BattleStatus   { return b.status }
func (b *Battle) Fund() int64            { return b.fund }

// --- мутации домена ---

func (b *Battle) Finish(status BattleStatus, finishedAt time.Time) {
	b.status = status
	b.finishedAt = &finishedAt
}

func (b *Battle) AddFund(delta int64) {
	if delta <= 0 {
		return
	}
	b.fund += delta
}
