// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration for the MySQL load-test API.
type Config struct {
	Port     string
	MySQLDSN string
}

// Load reads configuration from environment variables and returns Config.
// Required: MYSQL_DSN.
func Load() (Config, error) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		return Config{}, fmt.Errorf("MYSQL_DSN environment variable is required")
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Port:     port,
		MySQLDSN: dsn,
	}, nil
}
