// To run
// go run . / go run main.go
package main

import (
	"users-profile/configs"
	"users-profile/routes"

	"github.com/gin-gonic/gin"
)

var messi int

func main() {
	router := gin.Default()

	// Run database
	configs.ConnectDB()

	// Routes
	routes.UserRoute(router)

	router.Run()
}
