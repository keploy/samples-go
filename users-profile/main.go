// To run
// go run . / go run main.go
package main

import (
	"users-profile/configs"
	"users-profile/routes"

	"github.com/gin-gonic/gin"
	"github.com/keploy/go-sdk/integrations/kgin/v1"
	"github.com/keploy/go-sdk/keploy"
)

func main() {
	router := gin.Default()

	// Run database
	configs.ConnectDB()

	// Connecting keploy
	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "users-profile",
			Port: "8080",
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:8081/api",
		},
	})
	kgin.GinV1(k, router)

	// Routes
	routes.UserRoute(router)

	router.Run()
}
