package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

type User struct {
	id           uuid.UUID
	nickname     string
	role         UserRole
	xp           int
	createdAt    time.Time
	passwordHash string
}

// ----- КОНСТРУКТОР -----
func NewUser(id uuid.UUID, nickname, hashedPassword string) (*User, error) {
	if len(nickname) < 4 || len(nickname) > 20 {
		return nil, errors.New("nickname must be between 4 and 20 characters")
	}

	return &User{
		id:           id,
		nickname:     nickname,
		role:         RoleUser,
		xp:           0,
		createdAt:    time.Now(),
		passwordHash: string(hashedPassword),
	}, nil
}

func NewUserFromDB(
	id uuid.UUID,
	nickname string,
	role UserRole,
	xp int,
	createdAt time.Time,
	passwordHash string,
) (*User, error) {
	if len(nickname) < 4 || len(nickname) > 20 {
		return nil, errors.New("nickname must be between 4 and 20 characters")
	}
	if !role.IsValid() {
		return nil, errors.New("invalid user role")
	}

	return &User{
		id:           id,
		nickname:     nickname,
		role:         role,
		xp:           xp,
		createdAt:    createdAt,
		passwordHash: passwordHash,
	}, nil
}

// ----- ГЕТТЕРЫ -----
func (u *User) ID() uuid.UUID        { return u.id }
func (u *User) Nickname() string     { return u.nickname }
func (u *User) Role() UserRole       { return u.role }
func (u *User) XP() int              { return u.xp }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) PasswordHash() string { return u.passwordHash }

// ----- END ГЕТТЕРЫ -----

// ----- СЕТТЕРЫ -----
func (u *User) SetNickname(nickname string) error {
	if len(nickname) < 4 || len(nickname) > 20 {
		return errors.New("nickname must be between 4 and 20 characters")
	}
	u.nickname = nickname
	return nil
}

func (u *User) SetRole(role UserRole) error {
	if !role.IsValid() {
		return errors.New("invalid user role")
	}
	u.role = role
	return nil
}

// ----- END СЕТТЕРЫ -----

func (u *User) IsAdmin() bool {
	return u.role == RoleAdmin
}

func (u *User) AddXP(xp int) error {
	if xp < 0 {
		return errors.New("XP cannot be negative")
	}
	u.xp += xp
	return nil
}

// ----- ВЕРЕФИКАЦИЯ -----
func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte(password))
	return err == nil
}

// ----- END ВЕРЕФИКАЦИЯ -----

// ----- HELPERS -----
func (r UserRole) IsValid() bool {
	switch r {
	case RoleUser, RoleAdmin:
		return true
	default:
		return false
	}
}

func (r UserRole) String() string {
	return string(r)
}

// ----- END HELPERS -----
