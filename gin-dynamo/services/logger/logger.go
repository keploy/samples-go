package logger

import (
	"sync"
	"user-onboarding/config"

	"go.uber.org/zap"
)

var logger *zap.Logger
var once sync.Once

func Init() {
	once.Do(func() {
		if config.IsProduction() {
			logger, _ = zap.NewProduction()
		} else {
			logger, _ = zap.NewDevelopment()
		}
		defer logger.Sync()
	})
}

func Client() *zap.Logger {
	return logger
}
