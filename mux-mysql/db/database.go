package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/heyyakash/keploy-go-samples/models"
)

type Database struct {
	db *sql.DB
}

var Store *Database

func InstatiateDB() error {
	connStr := "root:my-secret-pw@tcp(127.0.0.1:3306)/mysql"
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	Store = &Database{
		db: db,
	}
	log.Printf("***Db is initialized*** \n")
	return nil
}

func (d *Database) IntializeTable() error {
	query := `create table if not exists list(
		id int auto_increment primary key,
		website varchar(400)
	)`
	_, err := d.db.Query(query)
	if err == nil {
		log.Printf("*** Tables created ***")
	}
	return err
}

func (d *Database) ReturnDB() *sql.DB {
	return d.db
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

func (d *Database) GetWebsiteFromId(id string) (string, error) {
	query := `select website from list where id=` + id
	var link string
	row := d.db.QueryRow(query)
	err := row.Scan(&link)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("Website not found")
		}
		return "", err
	}

	return link, nil
}

func (d *Database) GetAllLinks() ([]models.Table, error) {
	query := `select * from list`
	var array []models.Table
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var link models.Table
		if err := rows.Scan(&link.Id, &link.Website); err != nil {
			return nil, err
		}
		array = append(array, link)

	}
	return array, nil
}
