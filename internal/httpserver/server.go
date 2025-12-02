package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sundabaoh-rgb/tankionline/internal/auth"
	"github.com/sundabaoh-rgb/tankionline/internal/logger"
	"github.com/sundabaoh-rgb/tankionline/internal/match"
	"github.com/sundabaoh-rgb/tankionline/internal/room"
	"github.com/sundabaoh-rgb/tankionline/internal/ws"
)

type Server struct {
	addr         string
	log          logger.Logger
	http         *http.Server
	mux          *chi.Mux
	authService  auth.Service
	roomService  room.Service
	matchService *match.Service
	wsHub        *ws.Hub
}

func New(addr string,
	log logger.Logger,
	authSvc auth.Service,
	roomSvc room.Service,
	matchSvc *match.Service,
	wsHub *ws.Hub,
) *Server {
	mux := chi.NewRouter()

	s := &Server{
		addr:         addr,
		log:          log.Named("http"),
		mux:          mux,
		authService:  authSvc,
		roomService:  roomSvc,
		matchService: matchSvc,
		wsHub:        wsHub,
		http: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	s.setupRouter()

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
