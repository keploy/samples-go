package main

import (
	"database/sql"
	"fmt"
)

type ConnectionDetails struct {
	host     string // The hostname (default "localhost")
	port     string // The port to connect on (default "5438")
	user     string // The username (default "postgres")
	password string // The password (default "postgres")
	db_name  string // The database name (default "postgres")
}

func NewConnection(conn_details ConnectionDetails) (*sql.DB, error) {
	// Connect to PostgreSQL database
	db_info := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		conn_details.host, conn_details.port, conn_details.user, conn_details.password, conn_details.db_name)

	db, err := sql.Open("postgres", db_info)
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	defer db.Close()

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}
