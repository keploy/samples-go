// Package main starts the application
package main

import (
	"S3-Keploy/config"
	"S3-Keploy/routes"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	time.Sleep(2 * time.Second)
	awsService := config.Configuration()

	app := fiber.New()

	routes.Register(app, awsService)

	err := app.Listen(":3000")
	if err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}
}
