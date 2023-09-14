package main

import (
	"os"
	
	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	"go.uber.org/zap"
)

var col *mongo.Collection
var logger *zap.Logger

func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	errENV := godotenv.Load(".env")
	if errENV != nil {
		logger.Fatal("faild to load ENV Variable")
	}

	dbName, collection := "keploy", "url-shortener"

	client, err := New(os.Getenv("MONGO_URL"), dbName)
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

	r.Run(":" + port)
}
