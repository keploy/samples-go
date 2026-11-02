package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var jwtSecret = []byte("this is my secret")

type Todo struct {
	ID          int       `json:"id"`
	Task        string    `json:"task"`
	Progress    string    `json:"progress"` // Enum: Todo, InProgress, Done
	LastChecked time.Time `json:"last_checked"`
}

type TodoResponse struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	Todo      Todo      `json:"todo"`
}

type TodosResponse struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	Todos     []Todo    `json:"todos"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// Custom response writer to capture the status code for logging.
type responseWriter struct {
	http.ResponseWriter
	status int
}

var db *sql.DB

type contextKey string

var requestIDKey contextKey = "requestID"

func main() {
	var err error
	connStr := "user=user dbname=todo_db password=password sslmode=disable host=localhost port=5432"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'progress_status') THEN
				CREATE TYPE progress_status AS ENUM ('Todo', 'InProgress', 'Done');
			END IF;
		END $$;
	`)
	if err != nil {
		log.Fatalf("Unable to create progress_status enum: %v\n", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id SERIAL PRIMARY KEY,
			task TEXT NOT NULL,
			progress progress_status DEFAULT 'Todo',
			last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatalf("Unable to create table: %v\n", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS idempotency_keys (
			key TEXT PRIMARY KEY,
			todo_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatalf("Unable to create idempotency_keys table: %v\n", err)
	}

	r := mux.NewRouter()

	// Public routes
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.Use(requestIDMiddleware)
	r.Use(loggerMiddleware)

	// Create an API subrouter with middleware
	api := r.PathPrefix("/api").Subrouter()
	api.Use(jwtMiddleware)

	// Register all todo endpoints on the API subrouter, not on the main router
	api.HandleFunc("/todos", getTodos).Methods("GET")
	api.HandleFunc("/todos", createTodo).Methods("POST")
	api.HandleFunc("/todos/{id}", getTodo).Methods("GET")
	api.HandleFunc("/todos/{id}", updateTodo).Methods("PUT")
	api.HandleFunc("/todos/{id}", deleteTodo).Methods("DELETE")

	fmt.Println("Server is running on port 3040")
	log.Fatal(http.ListenAndServe(":3040", r))
}
