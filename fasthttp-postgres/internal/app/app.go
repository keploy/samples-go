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

	// Read database configuration from environment variables
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "db")

	// Build connection string from environment variables
	uri := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	log.Printf("Connecting to database at %s:%s/%s", dbHost, dbPort, dbName)

	db, err := sql.Open("postgres", uri)
	if err != nil {
		log.Print("Error connecting to database:", err)
		return err
	}

	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Println("Error closing database connection:", closeErr)
		}
	}()

	repo := repository.NewRepository(db)
	ctrl := handlers.NewHandler(repo)

	router := router.New()
	router.GET("/authors", ctrl.GetAllAuthors)
	router.GET("/books", ctrl.GetAllBooks)
	router.GET("/books/{id}", ctrl.GetBookByID)
	router.GET("/authors/{id}", ctrl.GetBooksByAuthorID)
	router.POST("/books", ctrl.CreateBook)
	router.POST("/authors", ctrl.CreateAuthor)

	server := &fasthttp.Server{
		Handler: router.Handler,
		Name:    "Server",
	}

	go func() {
		log.Println("Starting server: http://localhost:8080")
		if err := server.ListenAndServe(":8080"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")

	if err := server.Shutdown(); err != nil {
		log.Printf("Error shutting down server: %s\n", err)
		return err
	}

	log.Println("Server gracefully stopped")
	return nil
}

// getEnv reads an environment variable or returns a default value if not set
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
