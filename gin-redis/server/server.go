package server

import (
	"github.com/keploy/gin-redis/config"
	"github.com/keploy/gin-redis/routes"
)

func Init() {
	r := routes.NewRouter()
	r.Run(":" + config.Get().ServerPort) //running the server at port
}
