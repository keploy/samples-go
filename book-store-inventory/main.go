package main

import (
	"fmt"

	"github.com/saketV8/book-store-inventory/pkg/database"
	"github.com/saketV8/book-store-inventory/pkg/handlers"
	"github.com/saketV8/book-store-inventory/pkg/repositories"
	"github.com/saketV8/book-store-inventory/pkg/router"
	log "github.com/sirupsen/logrus"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.SetReportCaller(true)

	fmt.Println("============================================")
	fmt.Println("Book Store Inventory")
	fmt.Println("============================================")
	fmt.Println()

	// Setting up DB
	db, err := database.InitializeDatabase("sqlite3", "./DB/book_inventory.db")
	if err != nil {
		log.Fatal("DATABASE ERROR: ", err)
	}

	// METHOD 2
	// Initialize models
	// bookModel := &repositories.BookModel{DB: db}
	// you can add other model here

	// Initialize handlers
	// bookHandler := &handlers.BookHandler{BookModel: bookModel}
	// you can add other handler here

	// Initialize the app with all handlers
	// app := &router.App{
	// 	Books: bookHandler,
	// }

	// METHOD 1
	app := &router.App{
		Books: &handlers.BookHandler{
			BookModel: &repositories.BookModel{
				DB: db.DB,
			},
		},
	}

	router.SetupRouter(app)
}
