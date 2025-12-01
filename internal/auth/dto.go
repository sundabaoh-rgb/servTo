package auth

import (
	"time"

	"github.com/sundabaoh-rgb/tankionline/internal/domain"
)

type UserDTO struct {
	ID        string `json:"id"`
	Nickname  string `json:"nickname"`
	Role      string `json:"role"`
	XP        int    `json:"xp"`
	CreatedAt string `json:"created_at"`
}

func ToUserDTO(u *domain.User) UserDTO {
	return UserDTO{
		ID:        u.ID().String(),
		Nickname:  u.Nickname(),
		Role:      u.Role().String(),
		XP:        u.XP(),
		CreatedAt: u.CreatedAt().Format(time.RFC3339),
	}
}
