package httpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/sundabaoh-rgb/tankionline/internal/auth"
)

type contextKey string

const userIDKey contextKey = "userID"

func AuthMiddleware(authService auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(header, "Bearer ")

			userID, err := authService.ValidateAccessToken(token)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) any {
	return ctx.Value(userIDKey)
}
