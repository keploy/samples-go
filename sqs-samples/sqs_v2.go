//go:build !sqs_sdk_v1

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// The SQS consumer, with the AWS SDK for Go v2 (aws-sdk-go-v2): the default
// build. With go.mod, service/sqs speaks SQS's JSON protocol; with
// go.v2query.mod (go build -modfile=go.v2query.mod), service/sqs v1.26.0, the
// last release before that protocol, speaks the Query protocol: a form body
// (Action=ReceiveMessage&...&Version=2012-11-05) and XML answers.

// startSQSConsumer starts consuming the queue, telling rd when its first poll
// came back.
func startSQSConsumer(ctx context.Context, rd *readiness) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"), // LocalStack region (any valid region)
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "")), // LocalStack doesn't need real credentials
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc( //nolint:staticcheck
			func(service, _ string, _ ...interface{}) (aws.Endpoint, error) { //nolint:staticcheck
				// Pointing to the LocalStack SQS endpoint
				if service == sqs.ServiceID {
					return aws.Endpoint{ //nolint:staticcheck
						URL:           queueURL, // LocalStack SQS endpoint
						SigningRegion: "us-east-1",
					}, nil
				}
				return aws.Endpoint{}, fmt.Errorf("unknown service %s", service) //nolint:staticcheck
			},
		)),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
		return
	}

	fmt.Printf("awsCfg: %v\n", awsCfg)
	go consumeSQSMessages(ctx, awsCfg, rd)
}

// consumeSQSMessages consumes messages from the SQS queue, and tells rd when
// its first poll came back, or why it stopped.
func consumeSQSMessages(ctx context.Context, awsCfg aws.Config, rd *readiness) {
	logger, _ := zap.NewProduction()
	sqsClient := sqs.NewFromConfig(awsCfg)

	for {
		// Receive messages from SQS with WaitTimeSeconds = 4 and MaxMessages = 10
		resp, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &queueURL,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     4,
		})

		if err != nil {
			logger.Error(fmt.Sprintf("Failed to receive messages: %v", err))
			rd.markStopped(err)
			break
		}
		rd.markPolled()

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
				QueueUrl:      &queueURL,
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
