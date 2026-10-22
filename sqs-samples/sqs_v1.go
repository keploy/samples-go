//go:build sqs_sdk_v1

package main

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
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
		// Receive up to 10 messages, holding the poll 4 s on an empty queue
		// (0 in the forced variant: consumer.go).
		resp, err := sqsClient.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: aws.Int64(10),
			WaitTimeSeconds:     aws.Int64(int64(pollWaitSeconds())),
		})
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to receive messages: %v", err))
			rd.markStopped(err)
			consumerDeletes.notify() // no message's work is coming
			break
		}
		if forcedVariant() {
			// The forced variant is ready before any batch.
			rd.markPolled()
		}

		// If no messages were received, the app is ready, and polls again
		if len(resp.Messages) == 0 {
			logger.Info("No messages received")
			rd.markPolled()
			afterEmptyPoll(ctx)
			continue
		}

		// Process the received messages
		for _, message := range resp.Messages {
			event := message.Body
			log.Printf("Received event: %s", *event)

			// Record the event in MongoDB: a duplicate read, then the insert
			if err := recordEvent(context.Background(), *event); err != nil {
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
			// The message's work is over, its delete done or failed.
			consumerDeletes.notify()
		}
		// Ready once the batch's inserts and deletes are done, failed or
		// not: what the queue held as the app started is processed before
		// it says it is ready.
		rd.markPolled()
	}
}
