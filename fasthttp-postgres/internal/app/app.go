// Package app initializes the application, sets up database connections, routes, and handles server startup and graceful shutdown.
package app

import (
	"database/sql"
	"fasthttp-postgres/internal/handlers"
	"fasthttp-postgres/internal/repository"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"


	_ "github.com/lib/pq"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

func InitApp() error {
	time.Sleep(2 * time.Second)

	// Database connection initialization
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "password"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "db"
	}

	uri := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := sql.Open("postgres", uri)
	if err != nil {
		log.Print("Error connecting to database:", err)
		return err
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
		if err := server.ListenAndServe(":8080"); err != nil && err != http.ErrServerClosed {
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
		log.Printf("Error shutting down server: %s\n", err)
		return err
	}

	log.Println("Server gracefully stopped")
	return nil
}
