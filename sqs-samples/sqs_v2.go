//go:build !sqs_sdk_v1

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
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
		// Receive up to 10 messages, holding the poll 4 s on an empty queue
		// (0 in the forced variant: consumer.go).
		resp, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &queueURL,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     pollWaitSeconds(),
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
			_, err = sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      &queueURL,
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
