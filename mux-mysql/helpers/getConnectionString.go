package helpers

import (
	"log"
	"os"
)

func GetDBConnectionString() string {
	value := os.Getenv("ConnectionString")
	if value == "" {
		log.Println("Connection string empty")
		os.Exit(1)
	}
	return value
}
