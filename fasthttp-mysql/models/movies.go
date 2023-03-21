package models

import (
	"database/sql"
	"encoding/json"
)

func AddMovie(db *sql.DB, movie Movie) {
	_, err := db.Exec("INSERT INTO movies (title, year, rating) VALUES (?, ?, ?)", movie.Title, movie.Year, movie.Rating)
	if err != nil {
		panic(err.Error())
	}
}

func SingleMovie(db *sql.DB) (jsonMovie []byte) {
	var movie Movie
	if err := db.QueryRow("SELECT * FROM movies WHERE id = ?", countMovieID(db)).Scan(&movie.ID, &movie.Title, &movie.Year, &movie.Rating); err != nil {
		panic(err.Error())
	}
	jsonMovie, err := json.Marshal(movie)
	if err != nil {
		panic(err.Error())
	}
	return jsonMovie
}

func AllMovies(db *sql.DB) (jsonMovies []byte) {
	var (
		movie  Movie
		movies []Movie
	)
	rows, err := db.Query("SELECT * FROM movies")
	if err != nil {
		panic(err.Error())
	}
	for rows.Next() {
		err := rows.Scan(&movie.ID, &movie.Title, &movie.Year, &movie.Rating)
		if err != nil {
			panic(err.Error())
		}
		movies = append(movies, movie)
	}
	defer rows.Close()
	jsonMovies, err = json.Marshal(movies)
	if err != nil {
		panic(err.Error())
	}
	return jsonMovies
}

func countMovieID(db *sql.DB) int {
	var count int
	if err := db.QueryRow("SELECT COUNT(ID) FROM movies").Scan(&count); err != nil {
		panic(err.Error())
	}
	return count
}

// func AllMovieWithParameter(db *sql.DB, year, rating int) (jsonTitles []byte) {
// 	var (
// 		movie  Movie
// 		titles []string
// 	)
// 	yearRows, err := db.Query("SELECT * FROM movies WHERE year = ?", year)
// 	ratingRow, err := db.Query("SELECT * FROM movies WHERE rating = ?", rating)
// 	if err != nil {
// 		panic(err.Error())
// 	}
// 	for rows.Next() {
// 		err := rows.Scan(&movie.Title)
// 		if err != nil {
// 			panic(err.Error())
// 		}
// 		titles = append(titles, movie.Title)
// 	}
// 	defer rows.Close()
// 	jsonTitles, err = json.Marshal(titles)
// 	if err != nil {
// 		panic(err.Error())
// 	}

// 	return jsonTitles
// }
