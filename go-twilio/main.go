// Package main starts the application
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type Message struct {
	Body string `json:"Body" binding:"required"`
	To   string `json:"To" binding:"required"`
}

func sendSMS(c *gin.Context) {
	var message Message
	if err := c.ShouldBindJSON(&message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	twilioAccountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
	twilioPhoneNumber := os.Getenv("TWILIO_NUMBER")

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: twilioAccountSID,
		Password: twilioAuthToken,
	})

	params := &openapi.CreateMessageParams{}
	params.SetTo(message.To)
	params.SetFrom(twilioPhoneNumber)
	params.SetBody(message.Body)

	resp, err := client.Api.CreateMessage(params)
	if err != nil {
		log.Printf("Error sending SMS: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send SMS. Please check the provided phone number."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("SMS sent successfully! SID: %s", *resp.Sid)})
}

func gracefulShutdown(router *gin.Engine) {
	// Add any additional cleanup logic here if needed
	fmt.Println("Shutting down gracefully...")

	// Graceful shutdown logic
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to complete the ongoing requests
	if err := srv.Shutdown(context.TODO()); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	fmt.Println("Server exiting")
}

func main() {
	// Load environment variables from a .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	router := gin.Default()

	router.POST("/send-sms", sendSMS)

	// Run the server and listen for graceful shutdown
	go func() {
		if err := router.Run(":8080"); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	gracefulShutdown(router)
}
