package db

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

type Database struct {
	db *sql.DB
}

var Store *Database

func InstatiateDB() error {
	connStr := "root:my-secret-pw@tcp(127.0.0.1:3306)/"
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}

	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS test")
	db.Close()

	connStr = "root:my-secret-pw@tcp(127.0.0.1:3306)/test"

	db, err = sql.Open("mysql", connStr)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}

	Store = &Database{
		db: db,
	}
	return nil
}

func (d *Database) IntializeTable() error {
	query := `create table if not exists list(
		id int auto_increment primary key,
		website varchar(400)
	)`
	_, err := d.db.Query(query)
	return err
}

func (d *Database) EnterWebsiteToDB(link string) (int64, error) {
	query := `insert into list (website) values("` + link + `")`
	resp, err := d.db.Exec(query)
	if err != nil {
		return 0, err
	}
	lastInsertID, err := resp.LastInsertId()
	if err != nil {
		return 0, err
	}

	log.Printf("\n inserted %+v ", resp)
	return lastInsertID, err
}
