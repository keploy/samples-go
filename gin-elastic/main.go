package main

import (
	"context"
	"es4gophers/logic"

	"github.com/gin-gonic/gin"
	"github.com/keploy/go-sdk/integrations/kgin/v1"
	"github.com/keploy/go-sdk/keploy"
)

var ctx = context.Background()

func main() {
	port := "8080"
	// initialize keploy
	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "sample-elastic-search",
			Port: port,
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:8081/api",
		},
	})

	r := gin.Default()
	// integrate keploy with gin router
	kgin.GinV1(k, r)
	ctx = logic.ConnectWithElasticsearch(ctx)
	r.GET("/param", getURL)
	r.POST("/indexName", putURL)
	r.DELETE("/param", deleteURL)

	r.Run(":" + port)
}
