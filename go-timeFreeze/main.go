package main

import (
	"fmt"
	"log"
	"net/http"
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
	// Token expires in 2 minute.
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

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(tokenString))
}

func insecureExpiryOnlyMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func main() {
	http.HandleFunc("/login", loginHandler)

	http.Handle("/protected", insecureExpiryOnlyMiddleware(http.HandlerFunc(protectedHandler)))
	fmt.Println("current time :")
	fmt.Println(time.Now().Unix())
	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
