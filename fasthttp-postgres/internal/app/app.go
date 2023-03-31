package app

import (
	"context"
	"database/sql"
	"fasthttp-postgres/internal/handlers"
	"fasthttp-postgres/internal/repository"

	"log"

	"github.com/fasthttp/router"
	"github.com/keploy/go-sdk/integrations/kfasthttp"
	"github.com/keploy/go-sdk/integrations/ksql/v2"
	"github.com/keploy/go-sdk/keploy"
	"github.com/lib/pq"
	"github.com/valyala/fasthttp"
)

func InitApp() {
	driver := ksql.Driver{Driver: pq.Driver{}}
	sql.Register("keploy", &driver)

	uri := "postgresql://books_user:books_password@localhost:5433/books?sslmode=disable"
	db, err := sql.Open("keploy", uri)
	if err != nil {
		log.Fatal("Error connecting to database")
	}
	if err = db.PingContext(context.Background()); err != nil {
		log.Fatal("Ping doesn't work")
	}
	repo := repository.NewRepository(db)
	ctrl := handlers.NewHandler(repo)

	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "fasthttp-postgres",
			Port: "8080",
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:6789/api",
		},
	})
	mw := kfasthttp.FastHttpMiddleware(k)
	router := router.New()

	router.GET("/authors", ctrl.GetAllAuthors)
	router.GET("/books", ctrl.GetAllBooks)
	router.GET("/books/{id}", ctrl.GetBookById)
	router.GET("/authors/{id}", ctrl.GetBooksByAuthorId)
	router.POST("/books", ctrl.CreateBook)
	router.POST("/authors", ctrl.CreateAuthor)
	server := &fasthttp.Server{
		Handler: mw(router.Handler),
		Name:    "Server",
	}
	log.Println("Starting server: http://localhost:8080")
	if err := server.ListenAndServe(":8080"); err != nil {
		log.Fatalf("Error starting server: %s\n", err)
	}
}
