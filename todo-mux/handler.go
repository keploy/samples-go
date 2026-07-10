package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

func createTodo(w http.ResponseWriter, r *http.Request) {
	// Retrieve the request ID from the context
	requestID := r.Context().Value(requestIDKey).(string)

	// Check for Idempotency-Key header
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
		return
	}

	// Check if the request has already been processed
	var todoID int
	err := db.QueryRow(
		"SELECT todo_id FROM idempotency_keys WHERE key = $1",
		idempotencyKey,
	).Scan(&todoID)
	if err == nil {
		var todo Todo
		err = db.QueryRow(
			"SELECT id, task, progress, last_checked FROM todos WHERE id = $1",
			todoID,
		).Scan(&todo.ID, &todo.Task, &todo.Progress, &todo.LastChecked)
		if err != nil {
			log.Printf("Request ID: %s, Error: %v", requestID, err)
			http.Error(w, "Failed to fetch cached todo", http.StatusInternalServerError)
			return
		}

		response := TodoResponse{
			RequestID: requestID,
			Timestamp: time.Now(),
			Todo:      todo,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	} else if err != sql.ErrNoRows {
		log.Printf("Request ID: %s, Error: %v", requestID, err)
		http.Error(w, "Failed to check idempotency key", http.StatusInternalServerError)
		return
	}

	var todo Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		log.Printf("Request ID: %s, Error: %v", requestID, err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if todo.Progress != "Todo" && todo.Progress != "InProgress" && todo.Progress != "Done" {
		log.Printf("Request ID: %s, Error: Invalid progress value", requestID)
		http.Error(w, "Invalid progress value. Must be 'Todo', 'InProgress', or 'Done'", http.StatusBadRequest)
		return
	}

	err = db.QueryRow(
		"INSERT INTO todos (task, progress) VALUES ($1, $2) RETURNING id, last_checked",
		todo.Task, todo.Progress,
	).Scan(&todo.ID, &todo.LastChecked)
	if err != nil {
		log.Printf("Request ID: %s, Error: %v", requestID, err)
		http.Error(w, "Failed to create todo", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(
		"INSERT INTO idempotency_keys (key, todo_id) VALUES ($1, $2)",
		idempotencyKey, todo.ID,
	)
	if err != nil {
		log.Printf("Request ID: %s, Error: %v", requestID, err)
		http.Error(w, "Failed to store idempotency key", http.StatusInternalServerError)
		return
	}

	response := TodoResponse{
		RequestID: requestID,
		Timestamp: time.Now(),
		Todo:      todo,
	}

	// Send the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func getTodos(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(requestIDKey).(string)

	rows, err := db.Query("SELECT id, task, progress, last_checked FROM todos")
	if err != nil {
		http.Error(w, "Failed to fetch todos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.Task, &todo.Progress, &todo.LastChecked); err != nil {
			http.Error(w, "Failed to scan todo", http.StatusInternalServerError)
			return
		}
		todos = append(todos, todo)
	}

	response := TodosResponse{
		RequestID: requestID,
		Timestamp: time.Now(),
		Todos:     todos,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getTodo(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value(requestIDKey).(string)

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE todos SET last_checked = CURRENT_TIMESTAMP WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Failed to update last_checked", http.StatusInternalServerError)
		return
	}

	var todo Todo
	err = db.QueryRow(
		"SELECT id, task, progress, last_checked FROM todos WHERE id = $1", id,
	).Scan(&todo.ID, &todo.Task, &todo.Progress, &todo.LastChecked)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Todo not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch todo", http.StatusInternalServerError)
		}
		return
	}

	response := TodoResponse{
		RequestID: requestID,
		Timestamp: time.Now(),
		Todo:      todo,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var todo Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if todo.Progress != "Todo" && todo.Progress != "InProgress" && todo.Progress != "Done" {
		http.Error(w, "Invalid progress value. Must be 'Todo', 'InProgress', or 'Done'", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(
		"UPDATE todos SET task = $1, progress = $2 WHERE id = $3",
		todo.Task, todo.Progress, id,
	)
	if err != nil {
		http.Error(w, "Failed to update todo", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow(
		"SELECT id, task, progress, last_checked FROM todos WHERE id = $1", id,
	).Scan(&todo.ID, &todo.Task, &todo.Progress, &todo.LastChecked)
	if err != nil {
		http.Error(w, "Failed to fetch updated todo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM todos WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Failed to delete todo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate credentials (replace with your authentication logic)
	if loginReq.Username != "admin" || loginReq.Password != "password" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Create the JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": loginReq.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
	})

	// Sign the token with the secret key
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Send the token in the response
	response := LoginResponse{
		Token: tokenString,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
