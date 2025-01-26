// Package app initializes and runs the web server, sets up the database connection,
// and handles graceful shutdown of the server.
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

func InitApp(db *sql.DB) {
	time.Sleep(2 * time.Second)

	// Close the database connection when the application exits
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("%s", err)
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
