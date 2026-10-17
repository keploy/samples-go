// Package main implements an SQS sample app for Keploy testing.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

var queueURL = "http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/localstack-queue" // Your SQS queue URL

var col *mongo.Collection
var col2 *mongo.Collection
var logger *zap.Logger

func main() {
	logger, _ := zap.NewProduction()
	defer func() {
		err := logger.Sync() // flushes buffer, if any
		if err != nil {
			log.Println(err)
		}
	}()

	time.Sleep(2 * time.Second)

	dbName, collection := "keploy", "url-shortener"
	collection2 := "events"

	client, err := New("localhost:27017", dbName)
	if err != nil {
		logger.Fatal("failed to create mgo db client", zap.Error(err))
	}
	db := client.Database(dbName)

	col = db.Collection(collection)
	col2 = db.Collection(collection2)

	port := "8080"

	println("PID:", os.Getpid())

	ctx := context.Background()

	// /healthz's state, which the SQS consumer sets. The consumer is the AWS
	// SDK's this binary is built with (sqs_v2.go by default; sqs_v1.go with
	// -tags sqs_sdk_v1): see the README.
	ready := newReadiness()
	startSQSConsumer(ctx, ready)

	r := newRouter(ready)
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
	logger.Info("Server is listening on port 8080 , he")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("listen: %s\n", zap.Error(err))
	}
	logger.Info("Server is listening on port 8080")
	logger.Info("server exiting")
}

// newRouter is the app's routes; /healthz answers from rd.
func newRouter(rd *readiness) *gin.Engine {
	r := gin.Default()
	r.GET("/healthz", rd.healthz)
	r.GET("/:param", getURL)
	r.POST("/url", putURL)
	return r
}

// readiness is what /healthz answers: 503 until the SQS consumer's first
// ReceiveMessage came back, with messages or with none, and 200 from then on.
// A caller that waits on /healthz (keploy test's --health-url) then knows the
// app has already talked to SQS, however short its run. 200 means "the first
// poll is done", not "the consumer is alive": once 200, it stays 200.
type readiness struct {
	polled atomic.Bool
	// stopErr is why the consumer stopped, when it did.
	stopErr atomic.Pointer[error]
}

func newReadiness() *readiness {
	return &readiness{}
}

// markPolled records that a poll came back.
func (r *readiness) markPolled() {
	r.polled.Store(true)
}

// markStopped records that the consumer stopped on err.
func (r *readiness) markStopped(err error) {
	r.stopErr.Store(&err)
}

func (r *readiness) healthz(c *gin.Context) {
	if r.polled.Load() {
		c.String(http.StatusOK, "ok")
		return
	}
	if err := r.stopErr.Load(); err != nil {
		c.String(http.StatusServiceUnavailable, "SQS consumer stopped: %v", *err)
		return
	}
	c.String(http.StatusServiceUnavailable, "waiting for the first SQS poll")
}
