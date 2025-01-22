// Package main starts the application.
package main

import (
	"log"
)

func main() {
	a := &App{}
	err := a.Initialize()
	if err != nil {
		log.Fatal("Failed to initialize app", err)
	}
	log.Printf("😃 Connected to 8000 port !!")
	a.Run(":8000")
}
