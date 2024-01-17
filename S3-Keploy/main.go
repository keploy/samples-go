package main

import (
	"S3-Keploy/config"
	"S3-Keploy/routes"

	"github.com/gofiber/fiber/v2"
)

func main() {
	awsService := config.Configuration()

	app := fiber.New()

	routes.Register(app, awsService)

	app.Listen(":3000")
}
