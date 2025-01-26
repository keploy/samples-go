// Package main initializes and starts the application by calling the InitApp function
package main

import (
	"database/sql"
	"fasthttp-postgres/internal/app"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var (
	db  *sql.DB
	err error
)

func init() {
	uri := "postgresql://postgres:password@localhost:5432/db?sslmode=disable"
	db, err = sql.Open("postgres", uri)
	if err != nil {
		log.Printf("Error connecting to database: %s", err)
		os.Exit(1)
	}
}

func main() {
	app.InitApp(db)
}
