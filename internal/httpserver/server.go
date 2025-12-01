package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sundabaoh-rgb/tankionline/internal/auth"
	"github.com/sundabaoh-rgb/tankionline/internal/logger"
	"github.com/sundabaoh-rgb/tankionline/internal/room"
)

type Server struct {
	addr        string
	log         logger.Logger
	http        *http.Server
	mux         *chi.Mux
	authService auth.Service
	roomService room.Service
}

func New(addr string, log logger.Logger, authService auth.Service, roomService room.Service) *Server {
	mux := chi.NewRouter()

	s := &Server{
		addr:        addr,
		log:         log.Named("http"),
		mux:         mux,
		authService: authService,
		roomService: roomService,
		http: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	s.registerRoutes()

	return s
}

func (s *Server) Start() error {
	s.log.Info("starting http server", "addr", s.addr)
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("HTTP server shutting down")
	return s.http.Shutdown(ctx)
}

func (s *Server) Close() error {
	s.log.Info("HTTP server closing")
	return s.http.Close()
}
