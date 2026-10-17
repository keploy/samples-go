//go:build sqs_sdk_v1

package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// The SQS consumer, with the AWS SDK for Go v1 (aws-sdk-go), built with
// -tags sqs_sdk_v1. go.mod pins v1.47.5, the last release before SQS's JSON
// protocol: it speaks the Query protocol, a form body
// (Action=ReceiveMessage&...&Version=2012-11-05) and XML answers, as every
// application still on an aws-sdk-go v1 up to v1.47.5 does.

// startSQSConsumer starts consuming the queue, telling rd when its first poll
// came back.
func startSQSConsumer(ctx context.Context, rd *readiness) {
	// The LocalStack SQS endpoint queueURL is on.
	u, err := url.Parse(queueURL)
	if err != nil {
		log.Fatalf("unable to read the queue URL %q, %v", queueURL, err)
		return
	}
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"), // LocalStack region (any valid region)
		Endpoint:    aws.String(u.Scheme + "://" + u.Host),
		Credentials: credentials.NewStaticCredentials("dummy", "dummy", ""), // LocalStack doesn't need real credentials
	})
	if err != nil {
		log.Fatalf("unable to create an AWS session, %v", err)
		return
	}
	go consumeSQSMessages(ctx, sqs.New(sess), rd)
}

// consumeSQSMessages consumes messages from the SQS queue, and tells rd when
// its first poll came back, or why it stopped.
func consumeSQSMessages(ctx context.Context, sqsClient *sqs.SQS, rd *readiness) {
	logger, _ := zap.NewProduction()

	for {
		// Receive messages from SQS with WaitTimeSeconds = 4 and MaxMessages = 10
		resp, err := sqsClient.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: aws.Int64(10),
			WaitTimeSeconds:     aws.Int64(4),
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
			_, err = sqsClient.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL),
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
