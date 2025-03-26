package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

var queueUrl = "http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/localstack-queue" // Your SQS queue URL

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
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"), // LocalStack region (any valid region)
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "")), // LocalStack doesn't need real credentials
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				// Pointing to the LocalStack SQS endpoint
				if service == sqs.ServiceID {
					return aws.Endpoint{
						URL:           "http://localhost:4566", // LocalStack SQS endpoint
						SigningRegion: "us-east-1",
					}, nil
				}
				return aws.Endpoint{}, fmt.Errorf("unknown service %s", service)
			},
		)),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
		return
	}

	fmt.Printf("awsCfg: %v\n", awsCfg)

	go consumeSQSMessages(ctx, awsCfg)

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
	logger.Info("Server is listening on port 8080 , he")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("listen: %s\n", zap.Error(err))
	}
	logger.Info("Server is listening on port 8080")
	logger.Info("server exiting")
}

// consumeSQSMessages consumes messages from the SQS queue
func consumeSQSMessages(ctx context.Context, awsCfg aws.Config) {
	logger, _ := zap.NewProduction()
	sqsClient := sqs.NewFromConfig(awsCfg)

	for {
		// Receive messages from SQS with WaitTimeSeconds = 4 and MaxMessages = 10
		resp, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &queueUrl,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     4,
		})

		if err != nil {
			logger.Error(fmt.Sprintf("Failed to receive messages: %v", err))
			break
		}

		// If no messages were received, just continue
		if len(resp.Messages) == 0 {
			logger.Info("No messages received")
			continue
		}

		// Process the received messages
		for _, message := range resp.Messages {
			event := message.Body
			log.Printf("Received event: %s", *event)

			// Insert event into MongoDB
			eventDocument := bson.D{{Key: "event", Value: *event}, {Key: "processed_at", Value: time.Now()}}
			_, err := col2.InsertOne(context.Background(), eventDocument)
			if err != nil {
				log.Println("Failed to insert event into MongoDB:", err)
			}

			logger.Info(fmt.Sprintf("Received message: %s\n", *message.Body))
			// Delete the message after processing it
			_, err = sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      &queueUrl,
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to delete message: %v", err))
			} else {
				logger.Info(fmt.Sprintf("Message deleted: %s\n", *message.MessageId))
			}
		}
	}
}
