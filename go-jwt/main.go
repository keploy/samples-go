package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var (
	db     *gorm.DB
	err    error
	jwtKey = []byte("my_secret_key")
)

func initDB() {
	dsn := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	db, err = gorm.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	db.AutoMigrate(&User{})
}

func main() {
	initDB()
	defer db.Close()

	router := gin.Default()

	router.GET("/health", HealthCheckHandler)
	router.GET("/generate-token", GenerateTokenHandler)
	router.GET("/check-token", CheckTokenHandler)

	router.Run(":8000")
}
