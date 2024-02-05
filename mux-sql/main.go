// Package main starts the application
package main

import (
	"log"
)

func handleDeferError(err error) {
	if err != nil {
		log.Println(err)
	}
}

func main() {
	a := &App{}
	err := a.Initialize(
		"localhost", // postgres host
		// "postgres", //Change localhost to postgres when using Docker to run keploy
		"postgres", // username
		"password", // password
		"postgres") // db_name

	if err != nil {
		log.Fatal("Failed to initialize app", err)
	}

	log.Printf("😃 Connected to 8010 port !!")

	a.Run(":8010")
}
