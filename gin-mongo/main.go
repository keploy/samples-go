package main

import (
	"log"
	"time"

	"os"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	// "github.com/keploy/go-sdk/integrations/kgin/v1"
	// "github.com/keploy/go-sdk/integrations/kmongo"
	// "github.com/keploy/go-sdk/keploy"
	"go.uber.org/zap"
)

// var col *mongo.Collection
var col *mongo.Collection

var logger *zap.Logger

func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any

	dbName, collection := "keploy", "url-shortener"

	client, err := New("mongoDb:27017", dbName)
	if err != nil {
		logger.Fatal("failed to create mgo db client", zap.Error(err))
	}
	db := client.Database(dbName)

	col = db.Collection(collection)

	port := "8080"

	println("PID:", os.Getpid())

	r := gin.Default()

	r.GET("/:param", getURL)
	r.POST("/url", putURL)

	r.GET("/", get)
	go func() {
		log.Printf("Please wait. dont make api call now for mongo deps")
		time.Sleep(30 * time.Second)
		log.Printf("you can make api call now for mongo deps")
		client, err := New("localhost:27017", dbName)
		if err != nil {
			logger.Fatal("failed to create mgo db client", zap.Error(err))
		}
		db := client.Database(dbName)

		// integrate keploy with mongo
		// col = kmongo.NewCollection(db.Collection(collection))
		col = db.Collection(collection)
	}()
	r.Run(":" + port)

}
