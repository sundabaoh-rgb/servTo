package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Room struct {
	id              uuid.UUID
	name            string
	ownerID         uuid.UUID
	maxPlayers      int
	durationMinutes int
	goldPerKill     int
	fundModifier    float64
	createdAt       time.Time
}

func NewRoom(
	ownerID uuid.UUID,
	name string,
	maxPlayers int,
	durationMinutes int,
	goldPerKill int,
	fundModifier float64,
) (*Room, error) {
	if len(name) < 3 || len(name) > 50 {
		return nil, errors.New("room name must be between 3 and 50 characters")
	}
	if maxPlayers <= 0 {
		return nil, errors.New("maxPlayers must be positive")
	}
	if durationMinutes <= 0 {
		return nil, errors.New("durationMinutes must be positive")
	}
	if goldPerKill < 0 {
		return nil, errors.New("goldPerKill cannot be negative")
	}
	if fundModifier <= 0 {
		return nil, errors.New("fundModifier must be positive")
	}

	return &Room{
		id:              uuid.New(),
		name:            name,
		ownerID:         ownerID,
		maxPlayers:      maxPlayers,
		durationMinutes: durationMinutes,
		goldPerKill:     goldPerKill,
		fundModifier:    fundModifier,
		createdAt:       time.Now().UTC(),
	}, nil
}

// для чтения из БД
func NewRoomFromDB(
	id uuid.UUID,
	name string,
	ownerID uuid.UUID,
	maxPlayers int,
	durationMinutes int,
	goldPerKill int,
	fundModifier float64,
	createdAt time.Time,
) *Room {
	return &Room{
		id:              id,
		name:            name,
		ownerID:         ownerID,
		maxPlayers:      maxPlayers,
		durationMinutes: durationMinutes,
		goldPerKill:     goldPerKill,
		fundModifier:    fundModifier,
		createdAt:       createdAt,
	}
}

// Геттеры
func (r *Room) ID() uuid.UUID         { return r.id }
func (r *Room) Name() string          { return r.name }
func (r *Room) OwnerID() uuid.UUID    { return r.ownerID }
func (r *Room) MaxPlayers() int       { return r.maxPlayers }
func (r *Room) DurationMinutes() int  { return r.durationMinutes }
func (r *Room) GoldPerKill() int      { return r.goldPerKill }
func (r *Room) FundModifier() float64 { return r.fundModifier }
func (r *Room) CreatedAt() time.Time  { return r.createdAt }
