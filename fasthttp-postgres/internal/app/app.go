package app

import (
	"database/sql"
	"fasthttp-postgres/internal/handlers"
	"fasthttp-postgres/internal/repository"

	"log"

	"github.com/fasthttp/router"
	_ "github.com/lib/pq"
	"github.com/valyala/fasthttp"
)

func InitApp() {

	uri := "postgresql://books_user:books_password@localhost:5433/books?sslmode=disable"
	db, err := sql.Open("postgres", uri)
	if err != nil {
		log.Fatal("Error connecting to database")
	}
	repo := repository.NewRepository(db)
	ctrl := handlers.NewHandler(repo)
	router := router.New()

	router.GET("/authors", ctrl.GetAllAuthors)
	router.GET("/books", ctrl.GetAllBooks)
	router.GET("/books/{id}", ctrl.GetBookById)
	router.GET("/authors/{id}", ctrl.GetBooksByAuthorId)
	router.POST("/books", ctrl.CreateBook)
	router.POST("/authors", ctrl.CreateAuthor)
	server := &fasthttp.Server{
		Handler: router.Handler,
		Name:    "Server",
	}
	log.Println("Starting server: http://localhost:8080")
	if err := server.ListenAndServe(":8080"); err != nil {
		log.Fatalf("Error starting server: %s\n", err)
	}
}
