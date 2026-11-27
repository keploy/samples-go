// Package main implements a JWT-based HTTP server for Keploy time-freeze testing.
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

func loginHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Println("Login attempt at :", time.Now())
	expirationTime := time.Now().Add(10 * time.Second)

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
	if _, err = w.Write(jsonResponse); err != nil {
		log.Printf("write error: %v", err)
	}
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

		if claims.ExpiresAt.Before(time.Now()) {
			http.Error(w, fmt.Sprintf("Token is expired. Current timestamp: %d", time.Now().Unix()), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func protectedHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Welcome to the protected area!")); err != nil {
		log.Printf("write error: %v", err)
	}
}

const (
	// checkTimeTolerance is the maximum allowed skew between client and server
	// timestamps on /check-time.
	checkTimeTolerance = time.Second
	// unixMilliThreshold distinguishes Unix millisecond timestamps (13+ digits,
	// e.g. 1795763955968) from legacy Unix second timestamps (10 digits).
	unixMilliThreshold int64 = 1_000_000_000_000
)

// evaluateTimeSkew compares serverNow against a client-supplied Unix timestamp
// (milliseconds when >= unixMilliThreshold, otherwise legacy whole seconds) at
// matching precision so sub-second truncation never eats into the 1-second
// tolerance window.
func evaluateTimeSkew(serverNow time.Time, clientTimestamp int64) (serverVal int64, diff time.Duration, ok bool) {
	if clientTimestamp >= unixMilliThreshold {
		serverVal = serverNow.UnixMilli()
		diffMs := serverVal - clientTimestamp
		if diffMs < 0 {
			diffMs = -diffMs
		}
		diff = time.Duration(diffMs) * time.Millisecond
		return serverVal, diff, diff <= checkTimeTolerance
	}
	serverVal = serverNow.Unix()
	diffSec := serverVal - clientTimestamp
	if diffSec < 0 {
		diffSec = -diffSec
	}
	diff = time.Duration(diffSec) * time.Second
	return serverVal, diff, diff <= checkTimeTolerance
}

// checkTimeHandler checks if a client-provided timestamp is within 1 second of the server time.
func checkTimeHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Get the timestamp string from the URL query parameter 'ts'
	clientTimeStr := r.URL.Query().Get("ts")
	if clientTimeStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Missing 'ts' query parameter"}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	// 2. Parse the string into an integer (Unix timestamp in milliseconds or legacy seconds)
	clientTimestamp, err := strconv.ParseInt(clientTimeStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid timestamp format. Must be a Unix timestamp in milliseconds or seconds."}); err != nil {
			log.Printf("encode error: %v", err)
		}
		return
	}

	// 3. Compare server time against client time at matching precision
	serverVal, diff, ok := evaluateTimeSkew(time.Now(), clientTimestamp)

	fmt.Printf("Server Time: %d, Client Time: %d, Difference: %s\n", serverVal, clientTimestamp, diff)

	// 4. Check if the difference is greater than 1 second
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 5. If the check passes, wait for 1 second before sending a 200 OK response
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
