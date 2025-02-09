package database

import (
	"database/sql"
	"log"
)

type Database struct {
	DB *sql.DB
}

func InitializeDatabase(driver, dsn string) (*Database, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	log.Println("DB Connection Eastablished 🚀")

	return &Database{DB: db}, nil
}
