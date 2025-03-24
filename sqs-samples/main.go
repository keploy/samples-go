package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/joho/godotenv"
)

var queueUrl = "http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/localstack-queue" // Your SQS queue URL

func logger(msg string) {
	fmt.Println("[App]:", msg)
}

func main() {
	println("PID:", os.Getpid())

	err := godotenv.Load()
	if err != nil {
		logger("Warning: No .env file found, using environment variables.")
	}

	// accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	// secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	// region := os.Getenv("AWS_REGION")

	// if accessKey == "" || secretKey == "" || region == "" {
	// 	log.Fatal("Missing AWS credentials in environment variables")
	// }

	// Start fetching AppConfig and consuming SQS messages in separate goroutines
	appConf := NewAppConfig()

	ctx := context.Background()
	// awsCfg, err := config.LoadDefaultConfig(ctx)
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

	go appConf.fetchAppConfig(ctx, awsCfg)
	go consumeSQSMessages(ctx, awsCfg)

	// Wait for a termination signal
	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-stopper
	logger("Stopping the application...\n")
}

// consumeSQSMessages consumes messages from the SQS queue
func consumeSQSMessages(ctx context.Context, awsCfg aws.Config) {

	sqsClient := sqs.NewFromConfig(awsCfg)

	for {
		// Receive messages from SQS with WaitTimeSeconds = 4 and MaxMessages = 10
		resp, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &queueUrl,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     4,
		})

		if err != nil {
			logger(fmt.Sprintf("Failed to receive messages: %v", err))
			// continue
			break
		}

		// If no messages were received, just continue
		if len(resp.Messages) == 0 {
			logger("No messages received")
			continue
			// break
		}

		// Process the received messages
		for _, message := range resp.Messages {
			logger(fmt.Sprintf("Received message: %s\n", *message.Body))
			// Delete the message after processing it
			_, err := sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      &queueUrl,
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				logger(fmt.Sprintf("Failed to delete message: %v", err))
			} else {
				logger(fmt.Sprintf("Message deleted: %s\n", *message.MessageId))
			}
		}
	}
}
