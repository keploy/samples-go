// Package handlers contains the main bussiness logic of the application
package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/saketV8/book-store-inventory/pkg/models"
	"github.com/saketV8/book-store-inventory/pkg/repositories"
)

type BookHandler struct {
	BookModel *repositories.BookModel
}

func (bookHandler *BookHandler) GetBooksHandler(ctx *gin.Context) {
	books, err := bookHandler.BookModel.GetBooks()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get books list",
			"details": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, books)
}

func (bookHandler *BookHandler) GetBookByIDHandler(ctx *gin.Context) {
	bookIDParam := ctx.Param("book_id")
	bookIDParam = strings.TrimPrefix(bookIDParam, "/")

	if bookIDParam == "" || strings.TrimSpace(bookIDParam) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid or empty book ID in URL",
			"details": "Book ID must be a valid non-empty value",
		})
		return
	}

	// Only allow numeric IDs (if DB expects integers)
	matched, _ := regexp.MatchString(`^\d+$`, bookIDParam)
	if !matched {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Book ID",
			"details": "Book ID must be a positive integer",
		})
		return
	}

	book, err := bookHandler.BookModel.GetBookByID(bookIDParam)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Book not found",
				"details": "No book found with the given ID",
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get book by ID",
			"details": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, book)
}

func (bookHandler *BookHandler) AddBookHandler(ctx *gin.Context) {
	//getting param from POST request body
	// here body is of type models.Book
	var body models.Book

	// this will bind data coming from POST request to models.Book
	err := ctx.BindJSON(&body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Book Data",
			"details": err.Error(),
		})
		return
	}

	bookAdded, err := bookHandler.BookModel.AddBook(body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add book data",
			"details": err.Error(),
			"body":    body,
		})
		return
	}

	ctx.JSON(http.StatusOK, bookAdded)
}

func (bookHandler *BookHandler) DeleteBookHandler(ctx *gin.Context) {
	//getting param from POST request body
	// here body is of type models.Book
	var body models.BookDeleteRequestBody

	// this will bind data coming from POST request to models.Book
	err := ctx.BindJSON(&body)
	if err != nil {
		// If binding fails, return a 400 error with the error message
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Book ID",
			"details": err.Error(),
		})
		return
	}

	rowAffected, err := bookHandler.BookModel.DeleteBook(body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete book",
			"details": err.Error(),
			"body":    body,
		})
		return
	}

	// Add option to check if rowAffected == 0 then return Already Deleted or DATA DNE
	ctx.JSON(http.StatusOK, gin.H{
		"message":      "Book deleted successfully",
		"row-affected": rowAffected,
		"body":         body,
	})
}

func (bookHandler *BookHandler) UpdateBookHandler(ctx *gin.Context) {
	//getting param from POST request body
	// here body is of type models.Book
	var body models.BookUpdateRequestBody

	// this will bind data coming from POST request to models.Book
	err := ctx.BindJSON(&body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Book Data",
			"details": err.Error(),
		})
		return
	}

	rowAffected, err := bookHandler.BookModel.UpdateBook(body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update book",
			"details": err.Error(),
			"body":    body,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":      "Book updated successfully",
		"row-affected": rowAffected,
		"body":         body,
	})
}
