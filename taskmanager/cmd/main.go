package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"taskmanager/ent"
	handlers "taskmanager/handlers"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	client, err := ent.Open("postgres", "host=localhost port=5432 user=postgres password=new_password dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatalf("Failed opening connection to postgres: %v", err)
	}
	defer client.Close()

	// Run the auto migration tool.
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("Failed creating schema resources: %v", err)
	}

	taskHandler := handlers.NewTaskHandler(client)
	userHandler := handlers.NewUserHandler(client)

	r := mux.NewRouter()

	// Task routes
	r.HandleFunc("/tasks", taskHandler.GetAllTasks).Methods("GET")
	r.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.GetTask).Methods("GET")
	r.HandleFunc("/tasks", taskHandler.CreateTask).Methods("POST")
	r.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.UpdateTask).Methods("PUT")
	r.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.DeleteTask).Methods("DELETE")

	// User routes
	userRoutes := userHandler.UserRouter()
	r.PathPrefix("/users").Handler(userRoutes)

	// Start the server
	port := 8080
	fmt.Printf("Starting server on port %d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
