package main

import (
	"bytes"
	"encoding/json"
	"fasthttp-sql/models"

	"github.com/gookit/color"
	"github.com/valyala/fasthttp"
)

// GETindex handles HTTP GET requests and returns link to the documentation.
// It sets the content type to HTML, the status code to 200, and the body to a string
func GETindex(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/html")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte("Please refer to the documentation for the API: https://github.com/keploy/samples-go/tree/main/fasthttp-mysql"))
}

// POSTMovie handles HTTP POST requests for adds a new movie.
// It takes a request context and a database connection, then it decodes the request body into a movie
// struct, adds the movie to the database, and returns the movie as a JSON response
func POSTMovie(ctx *fasthttp.RequestCtx) {
	var movie models.Movie

	body := bytes.NewReader(ctx.PostBody())
	if err := json.NewDecoder(body).Decode(&movie); err != nil {
		color.Error.Tips("Invalid JSON: %s", err)
		ctx.Error("Invalid JSON", fasthttp.StatusBadRequest)
		return
	}

	models.AddMovie(ctx, movie)

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
// We're using the `ctx` object to get the `id` from the URL, then we're using that `id` to query the
// database for the movie with that `id`
func GETmovie(ctx *fasthttp.RequestCtx) {
	movie := models.SingleMovie(ctx)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(movie)
	
}

// GETAllMovies handles HTTP GET requests and returns all movies
// It returns all the movies in the database.
func GETAllMovies(ctx *fasthttp.RequestCtx) {
	movies := models.AllMovies(ctx)

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(movies)
}
