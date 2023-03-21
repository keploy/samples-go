package models

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
	"github.com/keploy/go-sdk/integrations/ksql/v1"
)


type Movie struct {
	ID     int    `json:"id" db:"id"`
	Title  string `json:"title" db:"title"`
	Year   int    `json:"year" db:"year"`
	Rating int    `json:"rating" db:"rating"`
}

func init() {
	// Register keploy sql driver to database/sql package. This should only be done once.
	driver := ksql.Driver{Driver: new(mysql.MySQLDriver)}
	sql.Register("keploy", &driver)
}


func Connect() *sql.DB {
	db, err := sql.Open("keploy", "root:root@/moviedb")
	if err != nil {
		panic(err)
	}
	return db
}
