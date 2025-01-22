// Package app initializes the application, sets up database connections, routes, and handles server startup and graceful shutdown.
package app

import (
	"database/sql"
	"fasthttp-postgres/internal/handlers"
	"fasthttp-postgres/internal/repository"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

func InitApp() {
	time.Sleep(2 * time.Second)
	// Database connection initialization
	uri := "postgresql://postgres:password@localhost:5432/db?sslmode=disable"
	db, err := sql.Open("postgres", uri)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	defer func() {
		// Close the database connection when the application exits
		if closeErr := db.Close(); closeErr != nil {
			log.Println("Error closing database connection:", closeErr)
		}
	}()

	repo := repository.NewRepository(db)
	ctrl := handlers.NewHandler(repo)

	// Router initialization
	router := router.New()
	router.GET("/authors", ctrl.GetAllAuthors)
	router.GET("/books", ctrl.GetAllBooks)
	router.GET("/books/{id}", ctrl.GetBookByID)
	router.GET("/authors/{id}", ctrl.GetBooksByAuthorID)
	router.POST("/books", ctrl.CreateBook)
	router.POST("/authors", ctrl.CreateAuthor)

	// Server initialization
	server := &fasthttp.Server{
		Handler: router.Handler,
		Name:    "Server",
	}

	// Start server in a goroutine
	go func() {
		log.Println("Starting server: http://localhost:8080")
		if err := server.ListenAndServe(":8080"); err != nil {
			log.Fatalf("Error starting server: %s\n", err)
		}
	}()

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // Notify on interrupt signals (e.g., Ctrl+C)

	// Block until a signal is received
	<-quit
	log.Println("Shutting down server...")

	// Attempt to gracefully shut down the server
	if err := server.Shutdown(); err != nil {
		log.Fatalf("Error shutting down server: %s\n", err)
	}

	log.Println("Server gracefully stopped")
}
