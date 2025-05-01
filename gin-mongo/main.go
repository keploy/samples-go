// Package main handle the server
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

var col *mongo.Collection
var logger *zap.Logger

func main() {
	time.Sleep(2 * time.Second)
	logger, _ := zap.NewProduction()
	defer func() {
		err := logger.Sync() // flushes buffer, if any
		if err != nil {
			log.Println(err)
		}
	}()

	dbName, collection := "keploy", "url-shortener"

	client, err := New("localhost:27017", dbName)
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
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopper

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Fatal("Server Shutdown:", zap.Error(err))
		}
		logger.Info("stopper called")
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("listen: %s\n", zap.Error(err))
	}
	logger.Info("server exiting")
}
