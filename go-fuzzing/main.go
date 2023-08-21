package main

import (
	"github.com/gin-gonic/gin"
	"github.com/keploy/go-sdk/integrations/kgin/v1"
	"github.com/keploy/go-sdk/keploy"
)

const port = "8080"

func integrateKeploy(r *gin.Engine) {
	// Initialize Keploy
	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "test-api",
			Port: port,
		},
		Server: keploy.ServerConfig{
			URL: "http://127.0.0.1:6789/api",
		},
	})

	kgin.GinV1(k, r)
}

func main() {
	router := gin.Default()

	// Integrate Keploy with Gin router
	integrateKeploy(router)

	router.POST("/validateInformation", validateInformation)

	// Run the server
	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}
