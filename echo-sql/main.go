// Package main handle the routes and initiates the server
package main

import (
	"database/sql"
	"os"

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

	defer handleDeferError(Database.Close())

	// init Keploy

	r := echo.New() // Init echo

	// kecho.EchoV4(k, r) // Tie echo router in with Keploy

	r.GET("/:param", GetURL)
	r.POST("/url", PutURL)
	r.DELETE("/:param", DeleteURL)
	r.PUT("/:param", UpdateURL)
	err = r.Start(":" + port)
	if err != nil {
		panic(err)
	}

}
