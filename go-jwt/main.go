// Package main is the entry point for the JWT-based user authentication service
// using Gin framework and PostgreSQL database. It provides endpoints for
// health check, token generation, and token validation.
package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var (
	db     *gorm.DB
	err    error
	jwtKey = []byte("my_secret_key")
)

// User represents a user in the database
type User struct {
	gorm.Model
	Username string `gorm:"unique"`
	Password string
	Token    string
}

// Claims struct
type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func initDB() {
	dsn := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	db, err = gorm.Open("postgres", dsn)
	if err != nil {
		log.Printf("Failed to connect to database: %s", err)
		os.Exit(1)
	}
	db.AutoMigrate(&User{})
}

// HealthCheckHandler handles the health check route
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// GenerateTokenHandler handles token generation
func GenerateTokenHandler(c *gin.Context) {
	// Normally, you'd get this from the request, but we're hardcoding it for simplicity
	username := "example_user"
	password := "example_password"

	// Set token expiration time
	expiryMinutesStr := c.DefaultQuery("expiry", "5")
	expiryMinutes, err := strconv.Atoi(expiryMinutesStr)
	if err != nil || expiryMinutes <= 0 {
		expiryMinutes = 5
	}
	expirationTime := time.Now().Add(time.Duration(expiryMinutes) * time.Minute)
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

	// Create or update the user
	var user User
	if db.Where("username = ?", username).First(&user).RecordNotFound() {
		user = User{Username: username, Password: password, Token: tokenString}
		db.Create(&user)
	} else {
		user.Password = password
		user.Token = tokenString
		db.Save(&user)
	}

	// Send the token to the client
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// CheckTokenHandler handles token verification
func CheckTokenHandler(c *gin.Context) {
	// Get the token from the request
	sentToken := c.Query("token")

	// Initialize a new instance of `Claims`
	claims := &Claims{}

	// Parse the JWT string and store the result in `claims`
	sentTokenObj, err := jwt.ParseWithClaims(sentToken, claims, func(_ *jwt.Token) (interface{}, error) {
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

	if !sentTokenObj.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Retrieve the user from the database
	var user User
	if db.Where("username = ?", claims.Username).First(&user).RecordNotFound() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Check if the sent token matches the stored token
	if user.Token != sentToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token does not match"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"username": claims.Username})
}

// CheckTimeHandler checks if a client-provided timestamp is within 1 second of the server time.
// The timestamp should be provided as a Unix timestamp in the 'ts' query parameter.
func CheckTimeHandler(c *gin.Context) {
	// 1. Get the timestamp string from the URL query parameter 'ts'
	clientTimeStr := c.Query("ts")
	if clientTimeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'ts' query parameter"})
		return
	}

	// 2. Parse the string into an integer (Unix timestamp)
	clientTimestamp, err := strconv.ParseInt(clientTimeStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp format. Must be a Unix timestamp in seconds."})
		return
	}

	// 3. Convert the integer timestamp to a time.Time object
	clientTime := time.Unix(clientTimestamp, 0)
	serverTime := time.Now()

	// 4. Calculate the duration (difference) between server time and client time
	diff := serverTime.Sub(clientTime)

	// 5. Get the absolute value of the duration, since the client could be ahead or behind
	if diff < 0 {
		diff = -diff
	}

	log.Printf(
		"Server Time: %s",
		serverTime.String(),
	)

	log.Printf(
		"Time difference: %s",
		diff.String(),
	)

	// 6. Check if the difference is greater than 1 second
	if diff > time.Second {
		c.Status(http.StatusBadRequest)
		return
	}

	time.Sleep(1 * time.Second)

	// 7. If the check passes, send a 200 OK response
	c.Status(http.StatusOK)
}

func main() {
	time.Sleep(2 * time.Second)
	initDB()
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("%s", err)
			os.Exit(1)
		}
	}()

	router := gin.Default()

	router.GET("/health", HealthCheckHandler)
	router.GET("/generate-token", GenerateTokenHandler)
	router.GET("/check-token", CheckTokenHandler)
	router.GET("/check-time", CheckTimeHandler)

	err = router.Run(":8000")
	if err != nil && err != http.ErrServerClosed {
		log.Printf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
