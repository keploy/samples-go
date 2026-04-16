// Package config handles configuration loading from environment variables.
package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	Port        string
	PostgresDSN string
}

func Load() (Config, error) {
	cfg := Config{
		Port:        getEnv("APP_PORT", "8080"),
		PostgresDSN: strings.TrimSpace(os.Getenv("POSTGRES_DSN")),
	}

	if cfg.PostgresDSN == "" {
		return Config{}, errors.New("POSTGRES_DSN is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
