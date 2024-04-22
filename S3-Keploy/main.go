// Package main starts the application
package main

import (
	"S3-Keploy/config"
	"S3-Keploy/routes"
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	awsService := config.Configuration()

	app := fiber.New()

	routes.Register(app, awsService)

	err := app.Listen(":3000")
	if err != nil {
		log.Fatal(err)
	}
}
