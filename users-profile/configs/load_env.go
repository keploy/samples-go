// Package configs implements the function that loads environment variable from .env file
package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// EnvMongoURI function that checks if the environment variable is correctly loaded and returns the environment variable.
func EnvMongoURI() string {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
		os.Exit(1)
	}

	return os.Getenv("MONGO-DB-URI")
}
