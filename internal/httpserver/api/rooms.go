package api

import "time"

type RoomResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	OwnerID         string    `json:"owner_id"`
	MaxPlayers      int       `json:"max_players"`
	DurationMinutes int       `json:"duration_minutes"`
	GoldPerKill     int       `json:"gold_per_kill"`
	FundModifier    float64   `json:"fund_modifier"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateRoomRequest struct {
	Name            string  `json:"name" validate:"required,min=3,max=50"`
	MaxPlayers      int     `json:"max_players" validate:"required,min=2,max=10"`
	DurationMinutes int     `json:"duration_minutes" validate:"required,min=5,max=60"`
	GoldPerKill     int     `json:"gold_per_kill" validate:"required,min=0,max=1000"`
	FundModifier    float64 `json:"fund_modifier" validate:"required,min=0.1,max=10.0"`
}

type JoinRoomRequest struct {
	RoomID string `json:"room_id" validate:"required,uuid4"`
}

type RoomPlayersResponse struct {
	Players []RoomPlayer `json:"players"`
}

type RoomPlayer struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Ready    bool   `json:"ready"`
}
