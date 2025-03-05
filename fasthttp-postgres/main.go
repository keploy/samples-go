// Package main is the entry point of the application.
package main

import (
	"fasthttp-postgres/internal/app"
	"log"
)

func main() {
	err := app.InitApp()
	if err != nil {
		log.Fatalf("Error occured: %s\n", err)
	}
}
