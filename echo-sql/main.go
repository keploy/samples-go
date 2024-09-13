// Package main handle the routes and initiates the server
package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var port = "8082"

var Logger *zap.Logger

var Database *sql.DB

func handleDeferError(err error) {
	if err != nil {
		Logger.Fatal(err.Error())
	}
}

func main() {
	time.Sleep(2 * time.Second)
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	var err error
	Logger, _ = zap.NewProduction()
	defer handleDeferError(Logger.Sync()) // flushes buffer

	Database, err = NewConnection(ConnectionDetails{
		host: "postgresDb",
		// host: "localhost" when using natively
		//host:     "echo-sql-postgres-1",
		port:     "5432",
		user:     "postgres",
		password: "password",
		dbName:   "postgres",
	})

	if err != nil {
		Logger.Fatal("Failed to establish connection to local PostgreSQL instance:", zap.Error(err))
	}

	defer Database.Close()

	// init Keploy

	r := echo.New() // Init echo

	// kecho.EchoV4(k, r) // Tie echo router in with Keploy

	r.GET("/:param", GetURL)
	r.POST("/url", PutURL)
	r.DELETE("/:param", DeleteURL)
	r.PUT("/:param", UpdateURL)

	// Create a channel to listen for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := r.Start(":" + port); err != nil && err != http.ErrServerClosed {
			Logger.Fatal("shutting down the server", zap.Error(err))
		}
	}()

	<-quit // Wait for OS signal to quit

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := r.Shutdown(ctx); err != nil {
		Logger.Fatal("Server forced to shutdown:", zap.Error(err))
	}

	Logger.Info("Server exiting")
}
