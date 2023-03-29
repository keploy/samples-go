package main

import (
	"log"
	"net/http"

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

	log.Fatal(http.ListenAndServe(":"+port, router))
}