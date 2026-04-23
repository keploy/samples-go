package config

import "os"

// Config holds runtime configuration for the gRPC load-test app.
type Config struct {
	HTTPPort string
	GRPCPort string
}

// Load reads configuration from environment variables.
func Load() *Config {
	httpPort := os.Getenv("APP_HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}
	grpcPort := os.Getenv("APP_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	return &Config{
		HTTPPort: httpPort,
		GRPCPort: grpcPort,
	}
}
