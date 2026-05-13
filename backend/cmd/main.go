package main

import (
	"attendance/config"
	"attendance/internal/logger"
)

func main() {
	// ======== config loader ========
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := logger.Setup(cfg)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	cfg.LogConfig(log)

	// ========  ========
}
