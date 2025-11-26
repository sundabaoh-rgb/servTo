package main

import (
	"log"

	"github.com/sundabaoh-rgb/tankionline/internal/app"
	"github.com/sundabaoh-rgb/tankionline/internal/config"
	"github.com/sundabaoh-rgb/tankionline/internal/logger"
)

func main() {
	// ----- READ CONFIG -----
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

	app.Test(logg)
	logg.Info("старый логгер вернулся")
}
