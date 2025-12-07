package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sundabaoh-rgb/tankionline/internal/httpserver/handlers"
	"github.com/sundabaoh-rgb/tankionline/internal/httpserver/middleware"
)

func (s *Server) setupRouter() {
	// Healthcheck
	s.mux.Get("/health", s.healthCheck)

	// Инициализация хендлеров
	authHandler := handlers.NewAuthHandler(s.authService)
	roomHandler := handlers.NewRoomHandler(s.roomService)

	// API Routes
	s.mux.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Group(func(public chi.Router) {
			public.Post("/register", authHandler.Register)
			public.Post("/login", authHandler.Login)
			public.Post("/refresh", authHandler.Refresh)

			s.mux.Get("/ws/game", s.handleGameWS)
		})

		// Protected routes
		r.Group(func(protected chi.Router) {
			protected.Use(middleware.Auth(s.authService))

			// Auth
			protected.Get("/me", authHandler.Me)
			protected.Post("/logout", authHandler.Logout)

			//battle

			// Rooms
			protected.Get("/rooms", roomHandler.List)
			protected.Post("/rooms", roomHandler.Create)
			protected.Post("/rooms/{roomID}/join", roomHandler.Join)
			protected.Post("/rooms/{roomID}/leave", roomHandler.Leave)
			protected.Get("/rooms/{roomID}/players", roomHandler.Players)
		})
	})

	// Static files for frontend
	s.mux.Handle("/*",
		http.StripPrefix("/",
			http.FileServer(http.Dir("./frontend")),
		),
	)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
