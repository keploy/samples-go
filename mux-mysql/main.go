package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/heyyakash/keploy-go-samples/controller"
)

var Port = ":8080"

func main() {
	store, err := CreateStore()
	if err != nil {
		log.Fatal("Couldnt create store ", err)
	}

	defer store.Close()
	router := mux.NewRouter()
	router.HandleFunc("/create", controller.CreateLink(store)).Methods("POST")
	router.HandleFunc("/all", controller.GetAllLinksFromWebsite(store)).Methods("GET")
	log.Print("Server is running")
	log.Fatal(http.ListenAndServe(Port, router))
}

func CreateStore() (*sql.DB, error) {
	connStr := "root:my-secret-pw@tcp(127.0.0.1:3306)/mysql"
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
	log.Print("*** DB Initiated ***")
	return store, nil
}
