package models

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/valyala/fasthttp"
)

// AddMovie takes a database connection and a movie object, and adds the movie to the database.
func AddMovie(ctx *fasthttp.RequestCtx, movie Movie) {
	if _, err := db.ExecContext(ctx, "INSERT INTO movies (title, year, rating) VALUES (?, ?, ?)", movie.Title, movie.Year, movie.Rating); err != nil {
		log.Fatal(err.Error())
	}
}

// SingleMovie returns the last movie that was added to the database.
// It takes a request context and a database connection, and returns a JSON encoded movie
func SingleMovie(ctx *fasthttp.RequestCtx) (jsonMovie []byte) {
	var (
		movie       Movie
		lastMovieID int
	)
	lastMovieID = countMovieID(ctx)
	lastMovieSQL := fmt.Sprintf("SELECT id, title, year, rating FROM movies WHERE id = %v", lastMovieID)
	if err := db.QueryRowContext(ctx, lastMovieSQL).Scan(&movie.ID, &movie.Title, &movie.Year, &movie.Rating); err != nil {
		log.Fatal(err.Error())
	}

	jsonMovie, err := json.Marshal(movie)
	if err != nil {
		log.Fatal(err.Error())
	}

	return jsonMovie
}

// AllMovies returns all movies in the database.
// It queries the database for all movies, scans the results into a slice of Movie structs, and then
// marshals the slice into a JSON array
func AllMovies(ctx *fasthttp.RequestCtx) (jsonMovies []byte) {
	var (
		movie  Movie
		movies []Movie
	)

	rows, err := db.QueryContext(ctx, "SELECT id, title, year, rating FROM movies")
	if err != nil {
		log.Fatal(err.Error())
	}

	for rows.Next() {
		err := rows.Scan(&movie.ID, &movie.Title, &movie.Year, &movie.Rating)
		if err != nil {
			log.Fatal(err.Error())
		}
		movies = append(movies, movie)
	}

	jsonMovies, err = json.Marshal(movies)
	if err != nil {
		log.Fatal(err.Error())
	}

	return jsonMovies
}

// countMovieID returns the number of movies in the database
// This function counts the number of rows in the movies table and returns the count as an integer.
func countMovieID(ctx *fasthttp.RequestCtx) int {
	var movie Movie
	if err := db.QueryRowContext(ctx, "SELECT COUNT(id) FROM movies;").Scan(&movie.ID); err != nil {
		log.Fatal(err.Error())
	}
	return movie.ID
}
