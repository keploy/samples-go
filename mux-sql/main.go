package main

import (
	"github.com/keploy/go-sdk/integrations/kmux"
	"github.com/keploy/go-sdk/keploy"
	"log"
)

func main() {
	a := &App{}
	err := a.Initialize(
		"postgres",
		"password",
		"postgres")

	if err != nil {
		log.Fatal("Failed to initialize app", err)
	}

	port := "8010"
	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "my-app",
			Port: port,
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:6789/api",
		},
	})

	a.Router.Use(kmux.MuxMiddleware(k))
	log.Printf("😃 Connected to 8010 port !!")

	a.Run(":8010")
}
