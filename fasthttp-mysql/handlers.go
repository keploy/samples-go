package main

import (
	"bytes"
	"encoding/json"
	"fasthttp-sql/models"

	"github.com/gookit/color"
	"github.com/valyala/fasthttp"
)

func GETindex(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/html")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte("Please refer to the documentation for the API: https://github.com/keploy/samples-go/tree/main/fasthttp-mysql"))
}

func POSTMovie(ctx *fasthttp.RequestCtx) {

	db := models.Connect()
	defer db.Close()
	body := bytes.NewReader(ctx.PostBody())
	var movie models.Movie
	if err := json.NewDecoder(body).Decode(&movie); err != nil {
		color.Error.Tips("Invalid JSON: %s", err)
		ctx.Error("Invalid JSON", fasthttp.StatusBadRequest)
		return
	}
	models.AddMovie(db, movie)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	byteMovie, err := json.Marshal(movie)
	if err != nil {
		color.Error.Tips("Parsing Error: %s", err)
		ctx.Error("Parsing Error", fasthttp.StatusInternalServerError)
		return
	}
	ctx.SetBody(byteMovie)
}

func GETmovie(ctx *fasthttp.RequestCtx) {

	db := models.Connect()
	defer db.Close()
	movie := models.SingleMovie(db)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(movie)
}

func GETAllMovies(ctx *fasthttp.RequestCtx) {

	db := models.Connect()
	defer db.Close()
	movies := models.AllMovies(db)
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
