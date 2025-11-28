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
	"github.com/sundabaoh-rgb/tankionline/internal/config"
	"github.com/sundabaoh-rgb/tankionline/internal/httpserver"
	"github.com/sundabaoh-rgb/tankionline/internal/logger"
	"github.com/sundabaoh-rgb/tankionline/internal/storage/postgres"
)

func main() {
	// ----- READ CONFIG -----
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		// Используем стандартный логгер для ошибок инициализации
		log.Fatalf("Failed to load config: %v", err)
	}
	// ----- END READ CONFIG -----

	// ----- CREATE LOGGER -----
	logg, err := logger.New(logger.Config{
		Mode: cfg.Env,
	})
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	logg = logg.Named("server")
	logg.Info("config loaded", "env", cfg.Env, "http_addr", cfg.HTTPAddr)
	defer logg.Sync() // не теряем логи
	// ----- END CREATE LOGGER -----

	// ----- START HTTP -----
	ctx, cancel := context.WithCancel(context.Background())
	pool, err := postgres.NewPool(ctx, cfg.PGDSN)
	if err != nil {
		logg.Error("failed to connect postgres", "err", err)
		return
	}
	defer pool.Close()
	logg.Info("connected to postgres")

	srv := httpserver.New(cfg.HTTPAddr, logg)
	//запускаем сервак в отдельном процессе
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) { //чтоб не ругалась на shutdown
			logg.Error("http server failed start", "server_error", err)
			cancel()
		}
	}()
	// ----- END HTTP -----

	// ----- SYSCALL to STOP -----
	sign := make(chan os.Signal, 1)
	signal.Notify(sign, syscall.SIGINT, syscall.SIGTERM)
	select {
	case stopNewContext := <-sign:
		logg.Info("catched syscall", "syscall", stopNewContext)
	case <-ctx.Done():
		logg.Info("starting shootdown")
	}
	// ----- END SYSCALL to STOP -----

	// ----- GRACEFUL SHUTDOWN -----
	logg.Info("starting graceful shutdown...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logg.Error("server shutdown error, starting forsed close", "err", err)

		if err := srv.Close(); err != nil {
			logg.Error("forced close failed, exiting anyway", "err", err)
		}
	}
	logg.Info("graceful shutdown completed")
	// ----- END GRACEFUL SHUTDOWN -----
}
