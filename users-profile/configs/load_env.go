// Creating a helper function to load the environment variable using the github.com/joho/godotenv
// GoDotEnv loads env vars from a .env file.
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
		log.Fatal("Error loading .env file")
	}

	return os.Getenv("MONGO-DB-URI")
}
