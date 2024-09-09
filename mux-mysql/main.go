// Package main start the application
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/heyyakash/keploy-go-samples/controller"
	"github.com/heyyakash/keploy-go-samples/helpers"
)

var Port = ":8080"

func main() {
	time.Sleep(1*time.Second)
	store, err := CreateStore()
	if err != nil {
		log.Fatal("Could not create store ", err)
	}

	defer func() {
		err = store.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/create", controller.CreateLink(store)).Methods("POST")
	router.HandleFunc("/all", controller.GetAllLinksFromWebsite(store)).Methods("GET")
	router.HandleFunc("/links/{id}", controller.RedirectUser(store)).Methods("GET")

	server := &http.Server{
		Addr:    Port,
		Handler: router,
	}

	// Run the server in a goroutine
	go func() {
		log.Print("Server is running on port", Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", Port, err)
		}
	}()

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	log.Println("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
	}

	if err := store.Close(); err != nil {
		log.Fatalf("Could not close database connection: %v\n", err)
	}

	log.Println("Server stopped")
}

func CreateStore() (*sql.DB, error) {
	log.Print(os.Getenv("DOCKER_ENV"))
	connStr := helpers.GetDBConnectionString()
	store, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, err
	}
	if err := store.Ping(); err != nil {
		return nil, err
	}
	query := `create table if not exists list(
		id int auto_increment primary key,
		website varchar(400)
	)`
	_, err = store.Exec(query)
	if err != nil {
		return nil, err
	}
	store.SetConnMaxLifetime(time.Hour)
	log.Print("*** DB Initiated at ***")
	return store, nil
}
