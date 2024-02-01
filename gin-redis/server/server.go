// Package server contains method to initiate the server
package server

import (
	"log"

	"github.com/keploy/gin-redis/routes"
)

func Init() {
	r := routes.NewRouter()
	port := "3001"
	err := r.Run(":" + port) //running the server at port
	if err != nil {
		log.Fatal(err)
	}
}
