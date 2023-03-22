package main

import (
	"bytes"
	"encoding/json"
	"fasthttp-sql/models"

	"github.com/gookit/color"
	"github.com/valyala/fasthttp"
)

// GETindex handles HTTP GET requests and returns link to the documentation.
func GETindex(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/html")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte("Please refer to the documentation for the API: https://github.com/keploy/samples-go/tree/main/fasthttp-mysql"))
}

// POSTMovie handles HTTP POST requests for adds a new movie.
func POSTMovie(ctx *fasthttp.RequestCtx) {
	var movie models.Movie

	body := bytes.NewReader(ctx.PostBody())
	if err := json.NewDecoder(body).Decode(&movie); err != nil {
		color.Error.Tips("Invalid JSON: %s", err)
		ctx.Error("Invalid JSON", fasthttp.StatusBadRequest)
		return
	}

	db := models.Connect()
	defer db.Close()
	models.AddMovie(ctx, db, movie)

	byteMovie, err := json.Marshal(movie)
	if err != nil {
		color.Error.Tips("Parsing Error: %s", err)
		ctx.Error("Parsing Error", fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(byteMovie)
}

// GETmovie handles HTTP GET requests and returns the last movie
func GETmovie(ctx *fasthttp.RequestCtx) {
	db := models.Connect()
	defer db.Close()
	movie := models.SingleMovie(ctx, db)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(movie)
}

// GETAllMovies handles HTTP GET requests and returns all movies
func GETAllMovies(ctx *fasthttp.RequestCtx) {
	db := models.Connect()
	defer db.Close()
	movies := models.AllMovies(ctx, db)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(movies)
}

// func GETmovies(ctx *fasthttp.RequestCtx) {
// 	db := models.Connect()
// 	defer db.Close()
// 	year := string(ctx.QueryArgs().Peek("year"))
// 	rating := string(ctx.QueryArgs().Peek("rating"))

// 	if year != "" {
// 		if _, err := strconv.Atoi(year); err == nil {
// 			fmt.Printf("year %q looks like a number.\n", year)
// 		} else {
// 			fmt.Printf("year %q looks like a string.\n", year)
// 		}
// 	}

// 	if rating != "" {
// 		if _, err := strconv.Atoi(rating); err == nil {
// 			fmt.Printf("ratings %q looks like a number.\n", rating)
// 		} else {
// 			fmt.Printf("ratings %q looks like a string.\n", rating)
// 		}
// 	}
// 	ctx.SetContentType("application/json")
// 	ctx.SetStatusCode(fasthttp.StatusOK)
// 	// ctx.SetBody(movies)
// }
