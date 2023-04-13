package manager

import (
	"github.com/coherentopensource/go-service-framework/util"
	"go.uber.org/zap"
	"log"
)

func mustInitLogger(cfg *Config) util.Logger {
	var logger *zap.Logger
	var err error
	switch cfg.Env {
	case EnvTest:
		logger = zap.NewNop()
	case EnvLocal:
		logger, err = zap.NewDevelopment()
		if err != nil {
			log.Fatalf("Failed to instantiate logger: %v", err)
		}
	default:
		logger, err = zap.NewProduction()
		if err != nil {
			log.Fatalf("Failed to instantiate logger: %v", err)
		}
	}

	return logger.Sugar()
}
