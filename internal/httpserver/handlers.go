package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) registerRoutes() {

	// Healthcheck
	s.mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})

	// ---------------------------
	//   AUTH ROUTES
	// ---------------------------
	authHandlers := NewAuthHandlers(s.authService)
	roomHandlers := NewRoomHandlers(s.roomService)

	s.mux.Route("/api/v1", func(r chi.Router) {
		// PUBLIC
		r.Post("/register", authHandlers.Register)
		r.Post("/login", authHandlers.Login)
		r.Post("/refresh", authHandlers.Refresh)

		// PROTECTED
		r.Group(func(pr chi.Router) {
			pr.Use(AuthMiddleware(s.authService))

			pr.Get("/me", authHandlers.Me)
			pr.Post("/logout", authHandlers.Logout)

			pr.Get("/rooms", roomHandlers.List)
			pr.Post("/rooms", roomHandlers.Create)
			pr.Post("/rooms/{roomID}/join", roomHandlers.Join)
			pr.Post("/rooms/{roomID}/leave", roomHandlers.Leave)
			pr.Get("/rooms/{roomID}/players", roomHandlers.Players)
		})
	})

	//front
	s.mux.Handle("/*",
		http.StripPrefix("/",
			http.FileServer(http.Dir("./frontend")),
		),
	)

}
