package main

import (
	"database/sql"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
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
	defer Logger.Sync()

	//	Database, err = NewConnection(ConnectionDetails{
	//		host:     "localhost",
	//		port:     "3306",
	//		user:     "user",
	//		password: "password",
	//		db_name:  "shorturl_db",
	//	})

	if err != nil {
		Logger.Fatal("Failed to establish connection to local MySQL instance:", zap.Error(err))
	}

	defer Database.Close()

	r := echo.New()

	r.GET("/:param", GetURL)
	r.POST("/url", PutURL)
	r.DELETE("/:param", DeleteURL)
	r.PUT("/:param", UpdateURL)
	err = r.Start(":" + port)
	if err != nil {
		panic(err)
	}
}
