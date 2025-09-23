package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Use a secure, randomly generated key in a real application.
// For this example, we'll use a simple byte slice.
var jwtKey = []byte("my_super_secret_key")

// Claims struct will be encoded to a JWT.
// We add a 'Username' field to the standard claims.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Handles the /login route.
// It generates and returns a new JWT for the user "testuser".
func loginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Login attempt at :", time.Now())
	// Token expires in 1 minute.
	expirationTime := time.Now().Add(5 * time.Minute)

	// Create the JWT claims, which includes the username and expiry time.
	claims := &Claims{
		Username: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	// Create a new token object, specifying signing method and the claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with our secret key to get the complete, signed token string.
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Finally, we send the token to the client.
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

		// DANGER: This parses the token but DOES NOT verify the signature.
		// It's like checking the date on a letter without checking the wax seal.
		// Anyone could have written this letter.
		_, _, err := new(jwt.Parser).ParseUnverified(tokenString, claims)
		if err != nil {
			http.Error(w, "Malformed token", http.StatusUnauthorized)
			return
		}

		// Manually check if the token is expired.
		// We are now trusting the expiry time from a token that could be a complete forgery.
		if claims.ExpiresAt.Time.Before(time.Now()) {
			http.Error(w, fmt.Sprintf("Token is expired. Current timestamp: %d", time.Now().Unix()), http.StatusUnauthorized)
			return
		}

		// If the expiry is "valid" (on this potentially fake token), let the request through.
		next.ServeHTTP(w, r)
	})
}

// Handles the /protected route.
// It simply responds with a welcome message if the JWT is valid.
func protectedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to the protected area!"))
}

func main() {
	// Route to get a token.
	http.HandleFunc("/login", loginHandler)

	// A protected route that requires a valid JWT.
	// We wrap the protectedHandler with our jwtMiddleware.
	http.Handle("/protected", insecureExpiryOnlyMiddleware(http.HandlerFunc(protectedHandler)))
	fmt.Println("current time :")
	fmt.Println(time.Now().Unix())
	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
