package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/sundabaoh-rgb/tankionline/internal/auth"
	"github.com/sundabaoh-rgb/tankionline/internal/config"
	"github.com/sundabaoh-rgb/tankionline/internal/httpserver"
	"github.com/sundabaoh-rgb/tankionline/internal/logger"
	"github.com/sundabaoh-rgb/tankionline/internal/match"
	"github.com/sundabaoh-rgb/tankionline/internal/room"
	"github.com/sundabaoh-rgb/tankionline/internal/storage/postgres"
	"github.com/sundabaoh-rgb/tankionline/internal/ws"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// -----------------------------------------------------------------------------
	// LOAD CONFIG (.env → config struct)
	// -----------------------------------------------------------------------------
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// -----------------------------------------------------------------------------
	// LOGGER INIT
	// -----------------------------------------------------------------------------
	logg, err := logger.New(logger.Config{
		Mode: cfg.Env,
	})
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	logg = logg.Named("server")
	logg.Info("config loaded", "env", cfg.Env, "http_addr", cfg.HTTPAddr)

	defer logg.Sync()

	// -----------------------------------------------------------------------------
	// GLOBAL CONTEXT
	// -----------------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// -----------------------------------------------------------------------------
	// POSTGRES CONNECTION
	// -----------------------------------------------------------------------------
	pool, err := postgres.NewPool(ctx, cfg.PGDSN)
	if err != nil {
		logg.Error("failed to connect postgres", "err", err)
	}
	defer pool.Close()

	logg.Info("connected to postgres")

	// -----------------------------------------------------------------------------
	// DEPENDENCIES (REPOS + SERVICES)
	// -----------------------------------------------------------------------------

	// --- User repository ---
	userRepo := postgres.NewUserRepo(pool)
	roomRepo := postgres.NewRoomRepo(pool)
	roomPlayersRepo := postgres.NewRoomPlayersRepo(pool)

	battleRepo := postgres.NewBattleRepo(pool)
	//battleStatsRepo := postgres.NewBattleStatsRepo(pool)

	// --- Token manager (in-memory) ---
	tokenManager := auth.NewMemoryTokenManager()

	// --- Password hasher (bcrypt) ---
	passwordHasher := auth.NewBcryptHasher(bcrypt.DefaultCost)

	// --- Auth service ---
	authService := auth.NewService(userRepo, tokenManager, passwordHasher, logg)

	roomService := room.NewService(roomRepo, roomPlayersRepo, logg)

	wsHub := ws.NewHub(logg)
	go wsHub.Run()

	matchService := match.NewService(battleRepo, wsHub, logg)
	// -----------------------------------------------------------------------------
	// HTTP SERVER + ROUTES
	// -----------------------------------------------------------------------------
	srv := httpserver.New(
		cfg.HTTPAddr,
		logg,
		authService,
		roomService,
		matchService,
		wsHub,
	)

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logg.Error("http server failed", "err", err)
			cancel()
		}
	}()

	// -----------------------------------------------------------------------------
	// SHUTDOWN LISTENER (SIGTERM / SIGINT)
	// -----------------------------------------------------------------------------
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stop:
		logg.Info("caught signal", "signal", sig)
	case <-ctx.Done():
		logg.Info("context canceled, stopping http server...")
	}

	// -----------------------------------------------------------------------------
	// GRACEFUL SHUTDOWN
	// -----------------------------------------------------------------------------
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	logg.Info("starting graceful shutdown")

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logg.Error("graceful shutdown failed, forcing close", "err", err)

		if err := srv.Close(); err != nil {
			logg.Error("forced close also failed", "err", err)
		}
	}

	logg.Info("shutdown complete, bye")
}
