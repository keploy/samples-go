package main

import (
	"database/sql"

	"github.com/keploy/go-sdk/integrations/kecho/v4"
	"github.com/keploy/go-sdk/keploy"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

const port = "8080"

var Logger *zap.Logger

var Database *sql.DB

func main() {
	var err error
	Logger, _ = zap.NewProduction()
	defer Logger.Sync() // flushes buffer

	Database, err = NewConnection(ConnectionDetails{
		host:     "localhost",
		port:     "5438",
		user:     "postgres",
		password: "postgres",
		db_name:  "postgres",
	})

	if err != nil {
		Logger.Fatal("Failed to establish connection to local PostgreSQL instance:", zap.Error(err))
	}

	defer Database.Close()

	// init Keploy
	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "sample-url-shortener",
			Port: port,
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:8081/api",
		},
	})

	r := echo.New() // Init echo

	kecho.EchoV4(k, r) // Tie echo router in with Keploy

	r.GET("/:param", GetURL)
	r.POST("/url", PutURL)

	r.Start(":" + port)

}
