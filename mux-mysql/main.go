package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/heyyakash/keploy-go-samples/controller"
	"github.com/heyyakash/keploy-go-samples/db"
)

func main() {
	if err := db.InstatiateDB(); err != nil {
		log.Fatal("Error Instantiating MySql Database ", err)
	}
	if err := db.Store.IntializeTable(); err != nil {
		log.Fatal("Couldn't create table", err)
	}
	router := mux.NewRouter()
	router.HandleFunc("/create", controller.CreateLink).Methods("POST")
	log.Print("Server is running")
	log.Fatal(http.ListenAndServe(":8080", router))
}
