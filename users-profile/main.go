// Package main starts the application.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
	"users-profile/configs"
	"users-profile/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	time.Sleep(2 * time.Second)
	// Initialize Gin router
	router := gin.Default()

	// Run database connection
	configs.ConnectDB()

	// Setup routes
	routes.UserRoute(router)

	// Create a HTTP server with a timeout
	server := &http.Server{
		Addr:    ":8080", // Replace with your desired port
		Handler: router,
	}

	// Start the server in a separate goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Setup graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt) // We can add more signals if needed

	// Block until a signal is received
	<-quit
	log.Println("Server shutting down...")

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown by shutting down the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}
