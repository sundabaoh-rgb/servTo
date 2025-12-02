package middleware

import (
	"net/http"
	"strings"

	"github.com/sundabaoh-rgb/tankionline/internal/auth"
	"github.com/sundabaoh-rgb/tankionline/internal/httpserver/response"
)

func Auth(authService auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				response.Unauthorized(w, "missing authorization token")
				return
			}

			token := strings.TrimPrefix(header, "Bearer ")

			userID, err := authService.ValidateAccessToken(token)
			if err != nil {
				response.Unauthorized(w, "invalid or expired token")
				return
			}

			ctx := SetUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
