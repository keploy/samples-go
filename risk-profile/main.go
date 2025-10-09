package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// --- V1 Data Structures and Data ---
type UserV1 struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// The base data remains a slice, but we will only serve the first element.
var originalUsers = []UserV1{
	{ID: 1, Name: "Alice", Email: "alice@example.com"},
	{ID: 2, Name: "Bob", Email: "bob@example.com"},
}

// --- V2 Data Structures for High-Risk Scenarios ---
type UserHighRiskTypeChange struct {
	ID    string `json:"id"` // Type changed from int to string
	Name  string `json:"name"`
	Email string `json:"email"`
}

// --- API Handlers ---

// BODY: LOW RISK (Only new fields added)
func getUsersLowRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if os.Getenv("KEPLOY_MODE") == "test" {
		dataWithAddedField := struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			Email    string `json:"email"`
			IsActive bool   `json:"isActive"` // New field
		}{
			ID: 1, Name: "Alice", Email: "alice@example.com", IsActive: true,
		}
		json.NewEncoder(w).Encode(dataWithAddedField)
		return
	}
	json.NewEncoder(w).Encode(originalUsers[0]) // Return a single object
}

// BODY: MEDIUM RISK (Value changes only)
func getUsersMediumRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if os.Getenv("KEPLOY_MODE") == "test" {
		dataWithValueChange := UserV1{
			ID: 1, Name: "Alicia", Email: "alice@example.com", // Name changed
		}
		json.NewEncoder(w).Encode(dataWithValueChange)
		return
	}
	json.NewEncoder(w).Encode(originalUsers[0]) // Return a single object
}

// BODY: MEDIUM RISK (New fields added + value changes)
func getUsersMediumRiskWithAddition(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if os.Getenv("KEPLOY_MODE") == "test" {
		dataWithAdditionAndChange := struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			Email    string `json:"email"`
			IsActive bool   `json:"isActive"` // New field
		}{
			ID: 1, Name: "Alicia", Email: "alice@example.com", IsActive: true, // Name changed AND IsActive added
		}
		json.NewEncoder(w).Encode(dataWithAdditionAndChange)
		return
	}
	json.NewEncoder(w).Encode(originalUsers[0]) // Return a single object
}

// BODY: HIGH RISK (Field's type changes)
func getUsersHighRiskType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if os.Getenv("KEPLOY_MODE") == "test" {
		dataWithTypeChange := UserHighRiskTypeChange{
			ID: "user-001", Name: "Alice", Email: "alice@example.com",
		}
		json.NewEncoder(w).Encode(dataWithTypeChange)
		return
	}
	json.NewEncoder(w).Encode(originalUsers[0]) // Return a single object
}

// BODY: HIGH RISK (Field is removed)
func getUsersHighRiskRemoval(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if os.Getenv("KEPLOY_MODE") == "test" {
		dataWithFieldRemoved := struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}{
			ID: 1, Name: "Alice",
		}
		json.NewEncoder(w).Encode(dataWithFieldRemoved)
		return
	}
	json.NewEncoder(w).Encode(originalUsers[0]) // Return a single object
}

// STATUS: HIGH RISK (Status code changes from 200 to 400)
func statusChangeHighRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	println("KEPLOY_MODE:", os.Getenv("KEPLOY_MODE"))
	if os.Getenv("KEPLOY_MODE") == "test" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Bad Request"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "OK"}`))
}

// HEADER: HIGH RISK (Content-Type changes)
func contentTypeChangeHighRisk(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("KEPLOY_MODE") == "test" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This is now plain text."))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "This is JSON."}`))
}

// HEADER: MEDIUM RISK (A non-critical header changes)
func headerChangeMediumRisk(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("KEPLOY_MODE") == "test" {
		w.Header().Set("X-Custom-Header", "new-value-987")
	} else {
		w.Header().Set("X-Custom-Header", "initial-value-123")
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "header test"}`))
}

// NOISY: This should PASS if noise is configured correctly.
func noisyHeader(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Date", time.Now().UTC().Format(http.TimeFormat))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Check the Date header!"}`))
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
	http.HandleFunc("/noisy-header", noisyHeader)
	port := "8080"
	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("could not start server: %s\n", err)
	}
}
