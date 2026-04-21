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

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/gin-gonic/gin"
)

var (
	pulsarClient   pulsar.Client
	producer       pulsar.Producer
	consumer       pulsar.Consumer
	receivedMsgs   []ReceivedMessage
)

const (
	topicName        = "persistent://public/default/sample-topic"
	subscriptionName = "sample-subscription"
)

type PublishRequest struct {
	Message string `json:"message" binding:"required"`
	Key     string `json:"key,omitempty"`
}

type ReceivedMessage struct {
	Key       string `json:"key,omitempty"`
	Payload   string `json:"payload"`
	MessageID string `json:"message_id"`
	Timestamp string `json:"timestamp"`
}

func main() {
	pulsarURL := os.Getenv("PULSAR_URL")
	if pulsarURL == "" {
		pulsarURL = "pulsar://localhost:6650"
	}

	var err error
	pulsarClient, err = pulsar.NewClient(pulsar.ClientOptions{
		URL:               pulsarURL,
		OperationTimeout:  30 * time.Second,
		ConnectionTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("Could not create Pulsar client: %v", err)
	}
	defer pulsarClient.Close()

	producer, err = pulsarClient.CreateProducer(pulsar.ProducerOptions{
		Topic: topicName,
	})
	if err != nil {
		log.Fatalf("Could not create producer: %v", err)
	}
	defer producer.Close()

	consumer, err = pulsarClient.Subscribe(pulsar.ConsumerOptions{
		Topic:            topicName,
		SubscriptionName: subscriptionName,
		Type:             pulsar.Shared,
	})
	if err != nil {
		log.Fatalf("Could not create consumer: %v", err)
	}
	defer consumer.Close()

	// Background consumer goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumeMessages(ctx)

	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/publish", publishHandler)
	r.GET("/messages", getMessagesHandler)

	srv := &http.Server{Addr: ":8090", Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

func publishHandler(c *gin.Context) {
	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := &pulsar.ProducerMessage{
		Payload: []byte(req.Message),
	}
	if req.Key != "" {
		msg.Key = req.Key
	}

	msgID, err := producer.Send(context.Background(), msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to publish: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "published",
		"message_id": fmt.Sprintf("%v", msgID),
	})
}

func getMessagesHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"count":    len(receivedMsgs),
		"messages": receivedMsgs,
	})
}

func consumeMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := consumer.Receive(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("Consumer receive error: %v", err)
				continue
			}
			consumer.Ack(msg)
			receivedMsgs = append(receivedMsgs, ReceivedMessage{
				Key:       msg.Key(),
				Payload:   string(msg.Payload()),
				MessageID: fmt.Sprintf("%v", msg.ID()),
				Timestamp: msg.PublishTime().Format(time.RFC3339),
			})
		}
	}
}
