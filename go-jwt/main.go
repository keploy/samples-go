// Package main starts the application
package main

import (
	"log"
	"net/http"
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
		log.Fatal("Failed to connect to database:", err)
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

func main() {
	time.Sleep(2 * time.Second)
	initDB()
	defer func() {
		if err := db.Close(); err != nil {
			log.Println("Error closing database connection:", err)
		}
	}()

	router := gin.Default()

	router.GET("/health", HealthCheckHandler)
	router.GET("/generate-token", GenerateTokenHandler)
	router.GET("/check-token", CheckTokenHandler)

	err = router.Run(":8000")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
