package models

import (
	"database/sql"
	"encoding/json"
	"log"

	"github.com/valyala/fasthttp"
)

func AddMovie(ctx *fasthttp.RequestCtx, db *sql.DB, movie Movie) {
	if _, err := db.ExecContext(ctx, "INSERT INTO movies (title, year, rating) VALUES (?, ?, ?)", movie.Title, movie.Year, movie.Rating); err != nil {
		log.Fatal(err.Error())
	}
}

func SingleMovie(ctx *fasthttp.RequestCtx, db *sql.DB) (jsonMovie []byte) {
	var (
		movie       Movie
		lastMovieID int
	)
	lastMovieID = countMovieID(ctx, db)
	if err := db.QueryRowContext(ctx, "SELECT * FROM movies WHERE id = ?", lastMovieID).Scan(&movie.ID, &movie.Title, &movie.Year, &movie.Rating); err != nil {
		log.Fatal(err.Error())
	}

	jsonMovie, err := json.Marshal(movie)
	if err != nil {
		log.Fatal(err.Error())
	}

	return jsonMovie
}

func AllMovies(ctx *fasthttp.RequestCtx, db *sql.DB) (jsonMovies []byte) {
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

func countMovieID(ctx *fasthttp.RequestCtx, db *sql.DB) int {
	var movie Movie
	if err := db.QueryRowContext(ctx, "SELECT COUNT(ID) FROM movies;").Scan(&movie.ID); err != nil {
		log.Fatal(err.Error())
	}
	return movie.ID
}

// func AllMovieWithParameter(db *sql.DB, year, rating int) (jsonTitles []byte) {
// 	var (
// 		movie  Movie
// 		titles []string
// 	)
// 	yearRows, err := db.Query("SELECT * FROM movies WHERE year = ?", year)
// 	ratingRow, err := db.Query("SELECT * FROM movies WHERE rating = ?", rating)
// 	if err != nil {
// 		 log.Fatal(err.Error())
// 	}
// 	for rows.Next() {
// 		err := rows.Scan(&movie.Title)
// 		if err != nil {
// 			 log.Fatal(err.Error())
// 		}
// 		titles = append(titles, movie.Title)
// 	}
// 	defer rows.Close()
// 	jsonTitles, err = json.Marshal(titles)
// 	if err != nil {
// 		 log.Fatal(err.Error())
// 	}

// 	return jsonTitles
// }
