package domain

import "github.com/google/uuid"

type PlayerInput struct {
	UserID uuid.UUID `json:"user_id"`
	Seq    uint64    `json:"seq"`

	Up    bool `json:"up"`
	Down  bool `json:"down"`
	Left  bool `json:"left"`
	Right bool `json:"right"`
	Shoot bool `json:"shoot"`
}
