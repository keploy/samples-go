package main

import (
	"database/sql"
	"fmt"

	"github.com/keploy/go-sdk/integrations/kecho/v4"
	"github.com/keploy/go-sdk/keploy"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

const port = "8080"

var logger *zap.Logger

const (
	host     = "localhost"
	db_port  = "5438"
	user     = "postgres"
	password = "postgres"
	dbname   = "postgres"
)

func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync() // flushes buffer

	// Connect to PostgreSQL database
	db_info := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, db_port, user, password, dbname)

	db, err := sql.Open("postgres", db_info)
	err = db.Ping()
	if err != nil {
		logger.Fatal("Failed to open PostgreSQL instance:", zap.Error(err))
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		logger.Fatal("Failed to establish connection to local PostgreSQL instance:", zap.Error(err))
	}

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

	r := echo.New()

	kecho.EchoV4(k, r)

	r.GET("/:param", nil)
	r.POST("/url", nil)

	r.Start(":" + port)

}
