// Package config handles configuration loading from environment variables.
package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	Port     string
	MongoURI string
}

func Load() (Config, error) {
	cfg := Config{
		Port:     getEnv("APP_PORT", "8080"),
		MongoURI: strings.TrimSpace(os.Getenv("MONGO_URI")),
	}

	if cfg.MongoURI == "" {
		return Config{}, errors.New("MONGO_URI is required")
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
