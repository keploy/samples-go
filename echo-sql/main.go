package main

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/keploy/go-sdk/integrations/kecho/v4"

	"github.com/keploy/go-sdk/keploy"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var port = "8082"

var Logger *zap.Logger

var Database *sql.DB

func main() {
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
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
	})

	r := echo.New() // Init echo

	// kecho.EchoV4(k, r) // Tie echo router in with Keploy
	r.Use(kecho.EchoMiddlewareV4(k))

	r.GET("/ritik", Get)
	r.GET("/:param", GetURL)
	r.POST("/url", PutURL)
	r.DELETE("/:param", DeleteURL)
	r.PUT("/:param", UpdateURL)
	err = r.Start(":" + port)
	if err != nil {
		panic(err)
	}

}

func Get(c echo.Context) error {
	c.JSON(http.StatusOK, map[string]string{
		"message": "ok",
	})
	return nil
}
