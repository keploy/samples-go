package models

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
	// _ "github.com/go-sql-driver/mysql"
	"github.com/keploy/go-sdk/integrations/ksql/v1"
)

// Movie is a struct that represents a movie.
// Movie is a struct with four fields, ID, Title, Year, and Rating, all of which are of type int except
// for Title which is of type string.
// @property {int} ID - The ID of the movie.
// @property {string} Title - The title of the movie.
// @property {int} Year - The year the movie was released
// @property {int} Rating - The rating of the movie on a scale of 1 to 10.
type Movie struct {
	ID     int    `json:"id" db:"id"`
	Title  string `json:"title" db:"title"`
	Year   int    `json:"year" db:"year"`
	Rating int    `json:"rating" db:"rating"`
}

var db *sql.DB

func init() {
	// Registering the keploy driver to the database/sql package.
	// Register keploy sql driver to database/sql package. This should only be done once.
	driver := ksql.Driver{Driver: new(mysql.MySQLDriver)}
	sql.Register("keploy", &driver)
	db = Connect()

	// https://stackoverflow.com/questions/39980902/golang-mysql-error-packets-go33-unexpected-eof
	// TL;DR connection timeout of server should be larger than client timeout
	db.SetMaxIdleConns(0)
}

// Connect returns a MySQL database connection object.
// Connect to the database and return a pointer to the database connection.
func Connect() *sql.DB {
	newConnection, err := sql.Open("keploy", "root:root@/moviedb")
	if err != nil {
		panic(err)
	}
	return newConnection
}
