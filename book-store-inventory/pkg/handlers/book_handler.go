package handlers

import (
	"net/http"

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

func (bookHandler *BookHandler) GetBookByIdHandler(ctx *gin.Context) {
	// getting param from the URL
	/* http://localhost:9090/api/v1/book/id/3 --> 3 is param here */
	book_id_param := ctx.Param("book_id")
	book, err := bookHandler.BookModel.GetBookByID(book_id_param)
	if err != nil {
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
