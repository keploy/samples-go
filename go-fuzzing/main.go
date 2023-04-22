package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/keploy/go-sdk/integrations/kgin/v1"
	"github.com/keploy/go-sdk/keploy"
)

// Book represents a book in our bookstore
type Book struct {
	ID     string `json:"id,omitempty"`
	Title  string `json:"title,omitempty"`
	Author string `json:"author,omitempty"`
}

var books []Book

// GetBooksHandler returns a list of all the books in the bookstore
func GetBooksHandler(c *gin.Context) {
	c.JSON(http.StatusOK, books)
}

// GetBookHandler returns a specific book based on its ID
func GetBookHandler(c *gin.Context) {
	for _, book := range books {
		if book.ID == c.Param("id") {
			c.JSON(http.StatusOK, book)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{})
}

// CreateBookHandler adds a new book to the bookstore
func CreateBookHandler(c *gin.Context) {
	var book Book
	if err := c.ShouldBindJSON(&book); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	books = append(books, book)
	c.JSON(http.StatusCreated, book)
}

// Create random strings of specified length
func randomString(length int) string {
	var result strings.Builder
	var charSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	for i := 0; i < length; i++ {
		randomIndex := rand.Intn(len(charSet))
		randomChar := string(charSet[randomIndex])
		result.WriteString(randomChar)
	}

	return result.String()
}

func main() {
	port := "8080"

	// initialize keploy
	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "sample-bookstore",
			Port: port,
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:6789/api",
		},
	})

	router := gin.Default()

	// integrate keploy with gin router
	kgin.GinV1(k, router)

	router.GET("/books", GetBooksHandler)
	router.GET("/books/:id", GetBookHandler)
	router.POST("/books", CreateBookHandler)

	// Fuzzing function
	go func() {
		for {
			// Generate random input data
			id := rand.Int()
			title := randomString(10)
			author := randomString(10)
			requestBody := fmt.Sprintf(`{"id":"%d","title":"%s","author":"%s"}`, id, title, author)

			// Create request with random data
			req, err := http.NewRequest(http.MethodPost, "http://localhost:"+port+"/books", bytes.NewBuffer([]byte(requestBody)))
			if err != nil {
				log.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Make request
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			// Log response status code if it's 400
			if resp.StatusCode == http.StatusBadRequest {
				log.Printf("fuzzing: status code = %d with data %s\n", resp.StatusCode, requestBody)
				break
			}
		}
	}()

	log.Fatal(http.ListenAndServe(":"+port, router))
}
