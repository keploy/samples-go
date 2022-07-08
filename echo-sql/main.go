package main

import (
	"database/sql"
	"os"

	"github.com/keploy/go-sdk/integrations/kecho/v4"
	"github.com/keploy/go-sdk/keploy"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var port = "8080"
var DBHost = "localhost"
var DBPort = "5438"
var KeployURL = "http://localhost:8081/api"

var Logger *zap.Logger

var Database *sql.DB

func main() {
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	if os.Getenv("DB_HOST") != "" {
		DBHost = os.Getenv("DB_HOST")
	}
	if os.Getenv("DB_PORT") != "" {
		DBPort = os.Getenv("DB_PORT")
	}
	if os.Getenv("KEPLOY_URL") != "" {
		KeployURL = os.Getenv("KEPLOY_URL")
	}
	var err error
	Logger, _ = zap.NewProduction()
	defer Logger.Sync() // flushes buffer

	Database, err = NewConnection(ConnectionDetails{
		host:     DBHost,
		port:     DBPort,
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
			URL: KeployURL,
		},
	})

	r := echo.New() // Init echo

	kecho.EchoV4(k, r) // Tie echo router in with Keploy

	r.GET("/:param", GetURL)
	r.POST("/url", PutURL)

	r.Start(":" + port)

}
