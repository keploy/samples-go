package main

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"log"
	"net/http"
	"time"
)

var (
	Db     *gorm.DB
	Err    error
	JwtKey = []byte("my_secret_key")
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

func InitDB() {
	dsn := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	Db, Err = gorm.Open("postgres", dsn)
	if Err != nil {
		log.Fatal("Failed to connect to database:", Err)
	}
	Db.AutoMigrate(&User{})
}

// HealthCheckHandler handles the health check route
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// GenerateTokenHandler handles token generation
func GenerateTokenHandler(c *gin.Context) {
	username := "example_user"
	password := "example_password"

	expirationTime := time.Now().Add(5 * time.Minute)

	claims := &Claims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(JwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	var user User
	if Db.Where("username = ?", username).First(&user).RecordNotFound() {
		user = User{Username: username, Password: password, Token: tokenString}
		Db.Create(&user)
	} else {
		user.Password = password
		user.Token = tokenString
		Db.Save(&user)
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// CheckTokenHandler handles token verification
func CheckTokenHandler(c *gin.Context) {
	sentToken := c.Query("token")

	claims := &Claims{}

	sentTokenObj, err := jwt.ParseWithClaims(sentToken, claims, func(token *jwt.Token) (interface{}, error) {
		return JwtKey, nil
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

	var user User
	if Db.Where("username = ?", claims.Username).First(&user).RecordNotFound() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if user.Token != sentToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token does not match"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"username": claims.Username})
}
