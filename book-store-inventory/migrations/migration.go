// Package main contains function for database migration
// It initialise the db with some default value
package main

// for database migration

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

func main() {
	// path where generated DB will be stored
	dbPath := filepath.Join("DB", "book_inventory.db")
	// path where migration file is present
	migrationsDir := "migrations"

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// defer db.Close()
	// fixing linter issue for defer

	// This is anonymous function :)
	defer func() {
		err := db.Close()
		if err != nil {
			log.Fatalf("Failed to close database: %v", err)
		}
	}()

	// Here Goose is being initialized and setting up the db type to sqlite
	err = goose.SetDialect("sqlite3")
	if err != nil {
		log.Fatalf("Failed to set Goose dialect: %v", err)
	}

	// Applying the migration
	err = goose.Up(db, migrationsDir)
	if err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	// Done :)
	fmt.Println("Migrations applied successfully!")
}
