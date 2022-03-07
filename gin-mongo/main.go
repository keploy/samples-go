package main

import (
	"github.com/gin-gonic/gin"
	"github.com/keploy/go-sdk/integrations/kgin/v1"
	"github.com/keploy/go-sdk/integrations/kmongo"
	"github.com/keploy/go-sdk/keploy"
	"go.uber.org/zap"
)

var col *kmongo.Collection
var logger *zap.Logger

func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any

	dbName, collection := "keploy", "url-shortener"
	client, err := New("localhost:27017", dbName)
	if err != nil {
		logger.Fatal("failed to create mgo db client", zap.Error(err))
	}
	db := client.Database(dbName)

	// integrate keploy with mongo
	col = kmongo.NewCollection(db.Collection(collection))

	port := "8080"
	// initialize keploy
	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "sample-url-shortener",
			Port: port,
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:8081/api",
		},
	})

	r := gin.Default()

	// integrate keploy with gin router
	kgin.GinV1(k, r)

	r.GET("/:param", getURL)
	r.POST("/url", putURL)

	r.Run(":" + port)
}
