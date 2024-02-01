// Package main start the application
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/heyyakash/keploy-go-samples/controller"
	"github.com/heyyakash/keploy-go-samples/helpers"
)

var Port = ":8080"

func main() {
	store, err := CreateStore()
	if err != nil {
		log.Fatal("Couldnt create store ", err)
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
	log.Print("Server is running")
	log.Fatal(http.ListenAndServe(Port, router))
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
