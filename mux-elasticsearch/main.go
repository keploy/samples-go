package main

import (
	"log"
	"os"
)

func main() {
	a := &App{}
	err := a.Initialize()
	if err != nil {
		log.Printf("Failed to initialize app %s", err)
		os.Exit(1)
	}
	log.Printf("😃 Connected to 8000 port !!")
	a.Run(":8000")
}
