// Package main starts the application. Command: go run .
package main

import (
	"log"
	"users-profile/configs"
	"users-profile/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Run database
	configs.ConnectDB()

	// Routes
	routes.UserRoute(router)

	err := router.Run()
	if err != nil {
		log.Fatal(err)
	}
}
