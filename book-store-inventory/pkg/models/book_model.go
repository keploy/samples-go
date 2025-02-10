package models

import "time"

type Book struct {
	BookID        int       `json:"book_id"`
	Title         string    `json:"title" binding:"required"`
	Author        string    `json:"author" binding:"required"`
	Price         float64   `json:"price" binding:"required,gte=0"`
	PublishedDate time.Time `json:"published_date" binding:"required"`
	StockQuantity int       `json:"stock_quantity" binding:"required,gte=0"`
}

type BookDeleteRequestBody struct {
	BookID int `json:"book_id" binding:"required"`
}

type BookUpdateRequestBody struct {
	Book
	// overriding BookID from Book Model, making it required field for updating the data
	BookID int `json:"book_id" binding:"required"`
}
