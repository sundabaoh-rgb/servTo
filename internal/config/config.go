package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env       string // тип логгера дев прод
	HTTPAddr  string
	PGDSN     string // коннект к бд
	TickRate  time.Duration
	XpPerKill int
}

// default values
const (
	envPrefix        = "TANKI_"
	defaultEnv       = "dev"
	defaultHTTPAddr  = ":8080"
	defaultPGDSN     = "plug" //заглушка
	defaultTickRate  = 30 * time.Millisecond
	defaultXpPerKill = 34
)

func getEnv(key, defoultValue string) string {
	fullKey := envPrefix + key
	if value := os.Getenv(fullKey); value == "" {
		return defoultValue
	} else {
		return value
	}
}

func isValidEnv(env string) bool {
	validEnvs := []string{"dev", "development", "prod", "production", "staging", "test"}
	for _, validEnv := range validEnvs {
		if strings.EqualFold(env, validEnv) {
			return true
		}
	}
	return false
}

func Load() (*Config, error) {
	cfg := &Config{}

	var err error

	// Environment
	cfg.Env = getEnv("ENV", defaultEnv)
	if !isValidEnv(cfg.Env) {
		return nil, fmt.Errorf("invalid environment: %s", cfg.Env)
	}

	// HTTP Address
	cfg.HTTPAddr = getEnv("HTTP_ADDR", defaultHTTPAddr)
	if cfg.HTTPAddr == "" {
		return nil, fmt.Errorf("HTTP address cannot be empty")
	}

	// PostgreSQL DSN
	cfg.PGDSN = getEnv("PG_DSN", defaultPGDSN) //! убрать заглушку defaultPGDSN
	if cfg.PGDSN == "" {
		return nil, fmt.Errorf("database DSN cannot be empty")
	}

	// Tick Rate
	tickRateStr := getEnv("TICK_RATE", "")
	if tickRateStr == "" {
		cfg.TickRate = defaultTickRate
	} else {
		tickRate, err := strconv.Atoi(tickRateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid tick rate: %w", err)
		}
		if tickRate <= 0 {
			return nil, fmt.Errorf("tick rate must be positive")
		}
		cfg.TickRate = time.Duration(tickRate) * time.Millisecond
	}

	// XP Per Kill
	xpStr := getEnv("XP_PER_KILL", "")
	if xpStr == "" {
		cfg.XpPerKill = defaultXpPerKill
	} else {
		cfg.XpPerKill, err = strconv.Atoi(xpStr)
		if err != nil {
			return nil, fmt.Errorf("invalid XP per kill: %w", err)
		}
		if cfg.XpPerKill <= 0 {
			return nil, fmt.Errorf("XP per kill must be positive")
		}
	}

	return cfg, nil
}
