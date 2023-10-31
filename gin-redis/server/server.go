package server

import (
	"github.com/keploy/gin-redis/routes"
)

func Init() {
	r := routes.NewRouter()
	port := "3001"
	r.Run(":" + port) //running the server at port
}
