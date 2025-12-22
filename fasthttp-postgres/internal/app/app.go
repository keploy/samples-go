// Package app initializes the application, sets up database connections, routes, and handles server startup and graceful shutdown.
package app

import (
	"database/sql"
	"fasthttp-postgres/internal/handlers"
	"fasthttp-postgres/internal/repository"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fasthttp/router"
	_ "github.com/lib/pq"
	"github.com/valyala/fasthttp"
)

func InitApp() error {
	time.Sleep(2 * time.Second)

	// Read DB config from environment variables
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbName == "" {
		log.Fatal("Database environment variables are not set")
	}

	uri := "postgresql://" + dbUser + ":" + dbPassword +
		"@" + dbHost + ":" + dbPort + "/" + dbName +
		"?sslmode=disable"

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

	// Router initialization
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
		log.Println("Starting server on :8080")
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
