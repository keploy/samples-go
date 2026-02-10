package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("my_super_secret_key")

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Login attempt at :", time.Now())
	expirationTime := time.Now().Add(2 * time.Minute)

	claims := &Claims{
		Username: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	response := map[string]string{"token": tokenString}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func insecureExpiryOnlyMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("time now :", time.Now().Unix())
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := authHeader[len("Bearer "):]
		claims := &Claims{}

		_, _, err := new(jwt.Parser).ParseUnverified(tokenString, claims)
		if err != nil {
			http.Error(w, "Malformed token", http.StatusUnauthorized)
			return
		}

		if claims.ExpiresAt.Time.Before(time.Now()) {
			http.Error(w, fmt.Sprintf("Token is expired. Current timestamp: %d", time.Now().Unix()), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to the protected area!"))
}

// checkTimeHandler checks if a client-provided timestamp is within 1 second of the server time.
func checkTimeHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Get the timestamp string from the URL query parameter 'ts'
	clientTimeStr := r.URL.Query().Get("ts")
	if clientTimeStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing 'ts' query parameter"})
		return
	}

	// 2. Parse the string into an integer (Unix timestamp in seconds)
	clientTimestamp, err := strconv.ParseInt(clientTimeStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid timestamp format. Must be a Unix timestamp in seconds."})
		return
	}

	// 3. Get the current server time as a Unix timestamp in seconds
	serverTimestamp := time.Now().Unix()

	// 4. Calculate the absolute difference in seconds
	diff := serverTimestamp - clientTimestamp
	if diff < 0 {
		diff = -diff
	}

	fmt.Printf("Server Time: %d, Client Time: %d, Difference: %ds\n", serverTimestamp, clientTimestamp, diff)

	// 5. Check if the difference is greater than 1 second
	if diff > 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 6. If the check passes, wait for 1 second before sending a 200 OK response
	time.Sleep(1 * time.Second)
	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/check-time", checkTimeHandler)

	http.Handle("/protected", insecureExpiryOnlyMiddleware(http.HandlerFunc(protectedHandler)))
	fmt.Println("current time :")
	fmt.Println(time.Now().Unix())
	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
