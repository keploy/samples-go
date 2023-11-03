package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/keploy/go-sdk/v2/keploy"
	"go.mongodb.org/mongo-driver/mongo"

	"go.uber.org/zap"
)

var col *mongo.Collection
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
	keploy.AddShutdownListener(func() {
		// fmt.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		//stop/shutdown the server and its routines.
		srv.Shutdown(ctx)
	})
	// go func() {
		if err := srv.ListenAndServe(); err == nil  {
			fmt.Printf("listen: %s\n", err)
		}
		fmt.Printf("listen: %s\n", err)

	// }()

	// Listen for system signals
	// quit := make(chan os.Signal)
	// signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// <-quit
	// srv.Shutdown(ctx)


}
