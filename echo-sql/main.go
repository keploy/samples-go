package main

import (
	"github.com/keploy/go-sdk/integrations/kecho/v4"
	"github.com/keploy/go-sdk/keploy"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const port = "8080"

var logger *zap.Logger

func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync() // flushes buffer

	// init Keploy
	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "sample-url-shortener",
			Port: port,
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:8081/api",
		},
	})

	r := echo.New()

	kecho.EchoV4(k, r)

	r.GET("/:param", nil)
	r.POST("/url", nil)

	r.Start(":" + port)

}
