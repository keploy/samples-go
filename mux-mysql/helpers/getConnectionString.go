package helpers

import (
	"log"
	"os"
)

func GetDBConnectionString() string {
	value := os.Getenv("ConnectionString")
	if value == "" {
		log.Fatal("Connection string empty")
	}
	return value
}
