// Package main provides a simple HTTP server that interacts with Elasticsearch to perform CRUD operations on documents.
// The server is built using the `gorilla/mux` router and `elasticsearch-go` client for communicating with the Elasticsearch service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gorilla/mux"
)

type App struct {
	Router *mux.Router
	DB     *elasticsearch.Client
	Server *http.Server
}

const SearchIndex = "documents"

// schemaNoiseDemo reports whether the keploy schema-noise-detection demo mode
// is enabled (env STAMP_CREATED_AT). It is OFF by default, so the app's normal
// behaviour and the committed keploy recordings are unchanged; the
// schema-noise-detection CI pipeline turns it on.
func schemaNoiseDemo() bool { return os.Getenv("STAMP_CREATED_AT") != "" }

func (a *App) Initialize() error {
	var err error
	cfg := elasticsearch.Config{}
	if schemaNoiseDemo() {
		// DisableKeepAlives so every outgoing Elasticsearch call uses a fresh
		// connection that closes right after the response. keploy finalizes an
		// outgoing HTTP mock when the connection closes; on a pooled keep-alive
		// connection it would otherwise buffer the request/response until the
		// app shuts down and lose them when the context is cancelled.
		cfg.Transport = &http.Transport{DisableKeepAlives: true}
	}
	// Addresses are taken from the ELASTICSEARCH_URL env when cfg.Addresses is
	// empty (NewClient honours it, same as NewDefaultClient).
	a.DB, err = elasticsearch.NewClient(cfg)

	if err != nil {
		return fmt.Errorf("error : %s", err)
	}

	_, err = esapi.IndicesExistsRequest{
		Index: []string{SearchIndex},
	}.Do(context.Background(), a.DB)

	if err != nil {
		fmt.Println("Indices is not present")
		a.CreateIndices()
	}

	a.Router = mux.NewRouter()
	a.Server = &http.Server{
		Addr:    ":8000",
		Handler: a.Router,
	}

	a.initializeRoutes()

	return nil
}

type Document struct {
	ID      string `json:"id,omitempty"`
	Title   string `json:"title"`
	Content string `json:"content"`
	// CreatedAt is stamped server-side on every create, so the document body
	// sent to Elasticsearch carries a volatile field that differs between a
	// keploy recording and each replay. This is what
	// `--schema-noise-detection` detects and persists as
	// req_body_noise: { body.created_at: [] } on the outgoing ES mock.
	CreatedAt int64 `json:"created_at,omitempty"`
}

type CreateDocumentResponse struct {
	ID string `json:"_id"`
}

func (a *App) CreateIndices() {
	var err error

	_, err = a.DB.Indices.Create(SearchIndex)

	if err != nil {
		fmt.Println(err)
	}
}

func (a *App) createDocument(w http.ResponseWriter, r *http.Request) {
	var doc Document
	err := json.NewDecoder(r.Body).Decode(&doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// In schema-noise demo mode, stamp a server-side creation time so the
	// outgoing ES request body drifts every run (the volatile field the
	// schema-noise-detection pipeline detects). Off by default.
	if schemaNoiseDemo() {
		doc.CreatedAt = time.Now().UnixNano()
	}

	data, err := json.Marshal(doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req := esapi.IndexRequest{
		Index: "documents",
		Body:  strings.NewReader(string(data)),
	}
	res, err := req.Do(context.Background(), a.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err = res.Body.Close(); err != nil {
			log.Printf("%s", err)
			os.Exit(1)
		}
	}()

	if res.IsError() {
		http.Error(w, res.String(), http.StatusInternalServerError)
		return
	}

	var createResponse CreateDocumentResponse
	if err := json.NewDecoder(res.Body).Decode(&createResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(map[string]string{"id": createResponse.ID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *App) getDocument(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	req := esapi.GetRequest{
		Index:      "documents",
		DocumentID: id,
	}
	res, err := req.Do(context.Background(), a.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		if err = res.Body.Close(); err != nil {
			log.Printf("%s", err)
			os.Exit(1)
		}
	}()

	if res.IsError() {
		http.Error(w, res.String(), http.StatusNotFound)
		return
	}

	var doc map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	source := doc["_source"].(map[string]interface{})

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *App) updateDocument(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	var doc Document
	err := json.NewDecoder(r.Body).Decode(&doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req := esapi.UpdateRequest{
		Index:      "documents",
		DocumentID: id,
		Body:       strings.NewReader(fmt.Sprintf(`{"doc": %s}`, string(data))),
	}
	res, err := req.Do(context.Background(), a.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err = res.Body.Close(); err != nil {
			log.Printf("%s", err)
			os.Exit(1)
		}
	}()

	if res.IsError() {
		http.Error(w, res.String(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *App) deleteDocument(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	req := esapi.DeleteRequest{
		Index:      "documents",
		DocumentID: id,
	}
	res, err := req.Do(context.Background(), a.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err = res.Body.Close(); err != nil {
			log.Printf("%s", err)
			os.Exit(1)
		}
	}()

	if res.IsError() {
		http.Error(w, res.String(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *App) Run(port string) {
	go func() {
		if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Could not listen on %s: %v\n", port, err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.Server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		os.Exit(1)
	}

	log.Println("Server exiting")
}

func (a *App) Hello(res http.ResponseWriter, _ *http.Request) {
	var result = "Hello"
	_, err := res.Write([]byte(result))
	if err != nil {
		http.Error(res, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/", a.Hello).Methods("GET")
	a.Router.HandleFunc("/documents", a.createDocument).Methods("POST")
	a.Router.HandleFunc("/documents/{id}", a.getDocument).Methods("GET")
	a.Router.HandleFunc("/documents/{id}", a.updateDocument).Methods("PUT")
	a.Router.HandleFunc("/documents/{id}", a.deleteDocument).Methods("DELETE")

}
