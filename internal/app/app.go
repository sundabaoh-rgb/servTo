package app

import "github.com/sundabaoh-rgb/tankionline/internal/logger"

type App struct {
	logger.Logger
}

func Test(log logger.Logger) {
	log = log.Named("App")
	log.Info("далао", "test", "test")
}
