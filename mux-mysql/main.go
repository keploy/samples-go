package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/heyyakash/keploy-go-samples/controller"
	"github.com/heyyakash/keploy-go-samples/db"
)

var Port = ":8080"

func main() {
	if err := db.InstatiateDB(); err != nil {
		log.Fatal("Error Instantiating MySql Database ", err)
	}
	if err := db.Store.IntializeTable(); err != nil {
		log.Fatal("Couldn't create table", err)
	}
	store := db.Store.ReturnDB()
	defer store.Close()
	router := mux.NewRouter()
	router.HandleFunc("/create", controller.CreateLink).Methods("POST")
	router.HandleFunc("/link/{id}", controller.RedirectUser).Methods("GET")
	router.HandleFunc("/all", controller.GetAllLinksFromWebsite).Methods("GET")
	log.Print("Server is running")
	log.Fatal(http.ListenAndServe(Port, router))
}
