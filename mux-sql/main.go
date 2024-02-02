package main

import (
	"log"
)

func main() {
	a := &App{}
	err := a.Initialize(
		"localhost", // postgres host
		//Change to `postgres` when using Docker to run keploy
		"postgres", // username
		"password", // password
		"postgres") // db_name

	if err != nil {
		log.Fatal("Failed to initialize app", err)
	}

	log.Printf("😃 Connected to 8010 port !!")

	a.Run(":8010")
}
