// Package main is the entry point for the pg-replicate HTTP server.
package main

import (
	"log"
	"net/http"

	"pg-replicate/db"
	"pg-replicate/handler"
)

func main() {
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close() //nolint:errcheck

	if err := db.Migrate(database); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	h := handler.New(database)

	http.HandleFunc("POST /companies", h.CreateCompany)
	http.HandleFunc("GET /companies", h.ListCompanies)
	http.HandleFunc("GET /companies/", h.GetCompany)

	http.HandleFunc("POST /projects", h.CreateProject)
	http.HandleFunc("GET /projects", h.ListProjects)
	http.HandleFunc("GET /projects/", h.GetProject)
	http.HandleFunc("PUT /projects/", h.UpdateProject)

	log.Println("server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
