package middleware

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "userID"

var (
	ErrUserIDNotFound = errors.New("user ID not found in context")
	ErrInvalidUserID  = errors.New("invalid user ID type in context")
)

func SetUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func GetUserID(ctx context.Context) (uuid.UUID, error) {
	val := ctx.Value(userIDKey)
	if val == nil {
		return uuid.Nil, ErrUserIDNotFound
	}

	userID, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrInvalidUserID
	}

	return userID, nil
}

func MustGetUserID(ctx context.Context) uuid.UUID {
	userID, err := GetUserID(ctx)
	if err != nil {
		panic("middleware: user ID not found in context - AuthMiddleware must be used")
	}
	return userID
}
