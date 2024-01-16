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

var messi int

func main() {
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	var err error
	Logger, _ = zap.NewProduction()
	defer Logger.Sync() // flushes buffer

	Database, err = NewConnection(ConnectionDetails{
		host: "postgresDb",
		//host:     "echo-sql-postgres-1",
		port:     "5432",
		user:     "postgres",
		password: "password",
		db_name:  "postgres",
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
	err = r.Start(":" + port)
	if err != nil {
		panic(err)
	}

}
