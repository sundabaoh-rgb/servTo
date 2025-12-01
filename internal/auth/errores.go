package auth

import "errors"

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrNicknameTaken = errors.New("nickname already taken")
	ErrInvalidInput  = errors.New("invalid input")
	ErrInvalidLogin  = errors.New("invalid credentials")
)
