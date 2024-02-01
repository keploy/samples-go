// Package db implements the functions to use database
package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/heyyakash/keploy-go-samples/models"
)

func EnterWebsiteToDB(link string, db *sql.DB) (int64, error) {
	query := `insert into list (website) values("` + link + `")`
	resp, err := db.Exec(query)
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

func GetWebsiteFromID(id string, db *sql.DB) (string, error) {
	query := `select website from list where id=` + id
	var link string
	row := db.QueryRow(query)
	err := row.Scan(&link)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("Website not found")
		}
		return "", err
	}

	return link, nil
}

func GetAllLinks(db *sql.DB) ([]models.Table, error) {
	query := `select * from list`
	var array []models.Table
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = rows.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	for rows.Next() {
		var link models.Table
		if err := rows.Scan(&link.ID, &link.Website); err != nil {
			return nil, err
		}
		array = append(array, link)

	}
	return array, nil
}
