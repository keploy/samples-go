package main

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

var jwtKey = []byte("my_secret_key")

// Claims struct
type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// HealthCheckHandler handles the health check route
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// GenerateTokenHandler handles token generation
func GenerateTokenHandler(c *gin.Context) {
	// Normally, you'd get this from the request, but we're hardcoding it for simplicity
	username := "example_user"

	// Set token expiration time
	expirationTime := time.Now().Add(5 * time.Minute)

	// Create the JWT claims, which includes the username and expiry time
	claims := &Claims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Create the token with the specified algorithm and claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	// Send the token to the client
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// CheckTokenHandler handles token verification
func CheckTokenHandler(c *gin.Context) {
	// Get the token from the request header
	tokenString := c.Query("token")

	// Initialize a new instance of `Claims`
	claims := &Claims{}

	// Parse the JWT string and store the result in `claims`
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token signature"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unauthorized"})
		return
	}

	if !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"username": claims.Username})
}

func main() {
	router := gin.Default()

	router.GET("/health", HealthCheckHandler)
	router.GET("/generate-token", GenerateTokenHandler)
	router.GET("/check-token", CheckTokenHandler)

	router.Run(":8000")
}
