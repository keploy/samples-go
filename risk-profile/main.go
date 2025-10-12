package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type UserV1 struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var originalUsers = []UserV1{
	{ID: 1, Name: "Alice", Email: "alice@example.com"},
}

func getUsersLowRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := originalUsers[0]
	response := map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"timestamp": time.Now().Unix(),
		"phone":     "9999988888",
	}
	json.NewEncoder(w).Encode(response)
}

func getUsersMediumRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := originalUsers[0]
	response := map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name + "-Modified",
		"email":     user.Email,
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func getUsersMediumRiskWithAddition(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := originalUsers[0]
	response := map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name + "-Modified",
		"email":     user.Email,
		"timestamp": time.Now().Unix(),
		"phone":     "9999988888",
	}
	json.NewEncoder(w).Encode(response)
}

func getUsersHighRiskType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := originalUsers[0]
	response := map[string]interface{}{
		"id":        "123",
		"name":      user.Name,
		"email":     user.Email,
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func getUsersHighRiskRemoval(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := originalUsers[0]
	response := map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name,
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func statusChangeHighRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	response := map[string]interface{}{
		"status":    "OK",
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func contentTypeChangeHighRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message":   "This is JSON.",
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func headerChangeMediumRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Custom-Header", "initial-value-456")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":    "header test",
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func statusBodyChange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	response := map[string]interface{}{
		"message":   "Status and body changed",
		"timestamp": time.Now().UnixNano(),
	}
	json.NewEncoder(w).Encode(response)
}

func headerBodyChange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Transaction-ID", "txn-2")
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message":   "Header and body changed",
		"timestamp": time.Now().UnixNano(),
	}
	json.NewEncoder(w).Encode(response)
}

func statusBodyHeaderChange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Transaction-ID", "txn-2")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	response := map[string]interface{}{
		"message":   "Status, body, and header changed",
		"timestamp": time.Now().UnixNano(),
	}
	json.NewEncoder(w).Encode(response)
}

func schemaCompletelyChanged(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "This is a completely different, non-JSON response body.")
}

func main() {
	log.Println("Application starting...")
	http.HandleFunc("/users-low-risk", getUsersLowRisk)
	http.HandleFunc("/users-medium-risk", getUsersMediumRisk)
	http.HandleFunc("/users-medium-risk-with-addition", getUsersMediumRiskWithAddition)
	http.HandleFunc("/users-high-risk-type", getUsersHighRiskType)
	http.HandleFunc("/users-high-risk-removal", getUsersHighRiskRemoval)
	http.HandleFunc("/status-change-high-risk", statusChangeHighRisk)
	http.HandleFunc("/content-type-change-high-risk", contentTypeChangeHighRisk)
	http.HandleFunc("/header-change-medium-risk", headerChangeMediumRisk)
	http.HandleFunc("/status-body-change", statusBodyChange)
	http.HandleFunc("/header-body-change", headerBodyChange)
	http.HandleFunc("/status-body-header-change", statusBodyHeaderChange)
	http.HandleFunc("/schema-completely-changed", schemaCompletelyChanged)
	port := "8080"
	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("could not start server: %s\n", err)
	}
}
