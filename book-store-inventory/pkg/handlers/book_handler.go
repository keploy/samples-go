// Package handlers contains the main bussiness logic of the application
package handlers

import (
	"fmt"
	"net/http"
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
	// getting param from the URL
	/* http://localhost:9090/api/v1/book/id/3 --> 3 is param here */
	bookIDParam := ctx.Param("book_id")

	// validation --> if book_id is empty ""
	if bookIDParam == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   fmt.Sprintf("Failed to get book by ID: %s", bookIDParam),
			"details": "Book ID cannot be empty",
		})
		return
	}

	book, err := bookHandler.BookModel.GetBookByID(bookIDParam)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   fmt.Sprintf("Failed to get book by ID: %s", bookIDParam),
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

	// If any field is missing in the body model then this bind function will take care of it
	// case of <""> is handles by BindJSON itself

	// this will bind data coming from POST request to models.Book
	err := ctx.BindJSON(&body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Book Data",
			"details": err.Error(),
		})
		return
	}

	// validation --> if any field is empty
	// strings.TrimSpace handles the case of --> <" "> ||  <"   "> || <"  ">
	if strings.TrimSpace(body.Title) == "" || strings.TrimSpace(body.Author) == "" || body.Price == 0 || body.PublishedDate.IsZero() || body.StockQuantity == 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add book data",
			"details": "Some fields are empty or invalid. Make sure all fields are filled properly.",
			"body":    body,
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
	var body models.BookDeleteRequestBody

	// Bind JSON body to struct, handles missing/empty fields automatically
	err := ctx.BindJSON(&body)
	if err != nil {
		// If binding fails, return a 400 error with the error message
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Book ID",
			"details": err.Error(),
		})
		return
	}

	// Deleting the book
	rowAffected, err := bookHandler.BookModel.DeleteBook(body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   fmt.Sprintf("Failed to delete the book by ID: %v", body.BookID),
			"details": err.Error(),
			"body":    body,
		})
		return
	}

	// Add option to check if rowAffected == 0 then return Already Deleted or DATA DNE(Does not exist)
	if rowAffected == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": fmt.Sprintf("Book not found or already deleted. ID: %v", body.BookID),
			"details": "No rows were affected.",
			"body":    body,
		})
		return
	}

	// if deletion is success
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

	// Bind JSON body to struct, handles missing/empty fields automatically
	err := ctx.BindJSON(&body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Book Data",
			"details": err.Error(),
		})
		return
	}

	// validation --> if any field is empty
	// strings.TrimSpace handles the case of --> <" "> ||  <"   "> || <"  ">
	if body.BookID == 0 || strings.TrimSpace(body.Title) == "" || strings.TrimSpace(body.Author) == "" || body.Price == 0 || body.PublishedDate.IsZero() || body.StockQuantity == 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update the book data",
			"details": "Some fields are empty or invalid. Make sure all fields are filled properly.",
			"body":    body,
		})
		return
	}

	rowAffected, err := bookHandler.BookModel.UpdateBook(body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   fmt.Sprintf("Failed to update the book data. ID: %v", body.BookID),
			"details": err.Error(),
			"body":    body,
		})
		return
	}

	// Add option to check if rowAffected == 0 then there's no update is done or that bookID does not exist
	if rowAffected == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": fmt.Sprintf("No book found to update. ID: %v", body.BookID),
			"details": "No rows were affected.",
			"body":    body,
		})
		return
	}

	// If successfully updated
	ctx.JSON(http.StatusOK, gin.H{
		"message":      "Book updated successfully",
		"row-affected": rowAffected,
		"body":         body,
	})
}
