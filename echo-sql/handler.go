package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/itchyny/base58-go"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

type ConnectionDetails struct {
	host     string // The hostname (default "localhost")
	port     string // The port to connect on (default "5438")
	user     string // The username (default "postgres")
	password string // The password (default "postgres")
	db_name  string // The database name (default "postgres")
}

type URLEntry struct {
	ID           string
	Redirect_URL string
	Created_At   time.Time
	Updated_At   time.Time
}

type urlRequestBody struct {
	URL string `json:"url"`
}

type successResponse struct {
	TS  int64  `json:"ts"`
	URL string `json:"url"`
}

/*
Establishes a connection with the PostgreSQL instance.
*/
func NewConnection(connDetails ConnectionDetails) (*sql.DB, error) {
	dbInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		connDetails.host, connDetails.port, connDetails.user, connDetails.password, connDetails.db_name)

	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		return nil, err
	}

	// Ping the database to ensure a connection can be established
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	// Check if the "url_map" table exists, create it if not
	if err := createURLMapTable(db); err != nil {
		return nil, err
	}

	return db, nil
}

func createURLMapTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS url_map (
			id           VARCHAR(255) PRIMARY KEY,
			redirect_url VARCHAR(255) NOT NULL,
			created_at   TIMESTAMP NOT NULL,
			updated_at   TIMESTAMP NOT NULL
		);
	`

	_, err := db.Exec(query)
	return err
}

func InsertURL(c context.Context, entry URLEntry) error {
	insert_query := `
		INSERT INTO url_map (id, redirect_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`

	select_query := `
			SELECT * 
			FROM url_map
			WHERE id = $1
	`

	update_timestamp := `
			UPDATE url_map
			SET updated_at = $1
			WHERE id = $2
	`

	res, err := Database.QueryContext(c, select_query, entry.ID) // See if the URL already exists
	if err != nil {
		return err
	}
	defer res.Close()
	if res.Next() { // If we get rows back, that means we have a duplicate URL
		var saved_entry URLEntry
		err := res.Scan(&saved_entry.ID, &saved_entry.Redirect_URL, &saved_entry.Created_At, &saved_entry.Updated_At)
		if err != nil {
			return err
		}

		_, err = Database.ExecContext(c, update_timestamp, entry.Updated_At, entry.ID)
		if err != nil {
			return err
		}

		return nil
	}

	_, err = Database.ExecContext(c, insert_query, entry.ID, entry.Redirect_URL, entry.Created_At, entry.Updated_At)
	if err != nil {
		return err
	}

	return nil
}

func GetURL(c echo.Context) error {
	err := Database.PingContext(c.Request().Context())
	if err != nil {
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not connect to Postgres.")
	}
	id := c.Param("param")

	find_url_query := `
		SELECT * 
		FROM url_map
		WHERE id = $1
	`
	res, err := Database.QueryContext(c.Request().Context(), find_url_query, id) // Attempt to find a matching URL

	if err != nil {
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "Error encountered while attempting to lookup URL.")
	}
	defer res.Close()
	if !res.Next() {
		return echo.NewHTTPError(http.StatusNotFound, "Invalid URL ID.")
	}

	var entry URLEntry
	err = res.Scan(&entry.ID, &entry.Redirect_URL, &entry.Created_At, &entry.Updated_At)
	if err != nil {
		Logger.Fatal(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not successfully scan retrieved DB entry.")
	}

	c.Redirect(http.StatusPermanentRedirect, entry.Redirect_URL)
	return nil
}

func DeleteURL(c echo.Context) error {
	err := Database.PingContext(c.Request().Context())
	if err != nil {
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not connect to Postgres.")
	}
	id := c.Param("param")

	sqlStatement := `
	DELETE FROM url_map
	WHERE id = $1;`
	res, err := Database.ExecContext(c.Request().Context(), sqlStatement, id)
	if err != nil {
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "Error encountered while attempting to delete URL.")
	}

	rowsAffected, _ := res.RowsAffected()
	return c.JSON(http.StatusOK, map[string]int64{
		"rows_affected": rowsAffected,
	})
}

func UpdateURL(c echo.Context) error {
	err := Database.PingContext(c.Request().Context())
	if err != nil {
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not connect to Postgres.")
	}
	id := c.Param("param")

	req_body := new(urlRequestBody)
	err = c.Bind(req_body)
	if err != nil {
		fmt.Println(req_body)
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to decode request.")
	}

	u := req_body.URL

	if u == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing URL parameter")
	}

	select_query := `
			SELECT * 
			FROM url_map
			WHERE id = $1
	`

	update_timestamp := `
			UPDATE url_map
			SET updated_at = $1, redirect_url = $2
			WHERE id = $3
	`

	res, err := Database.QueryContext(c.Request().Context(), select_query, id) // See if the URL already exists
	if err != nil {
		return err
	}
	defer res.Close()
	if res.Next() { // If we get rows back, that means we have a duplicate URL
		var saved_entry URLEntry
		err := res.Scan(&saved_entry.ID, &saved_entry.Redirect_URL, &saved_entry.Created_At, &saved_entry.Updated_At)
		if err != nil {
			return err
		}
		result, err := Database.ExecContext(c.Request().Context(), update_timestamp, time.Now(), u, id)
		if err != nil {
			return err
		}

		rowsAffected, _ := result.RowsAffected()
		return c.JSON(http.StatusOK, map[string]int64{
			"rows_affected": rowsAffected,
		})
	}
	return c.JSON(http.StatusOK, map[string]string{
		"error": "no document exists with given id",
	})
}

func PutURL(c echo.Context) error {

	err := Database.PingContext(c.Request().Context())
	if err != nil {
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not connect to Postgres.")
	}

	req_body := new(urlRequestBody)

	err = c.Bind(req_body)
	if err != nil {
		fmt.Println(req_body)
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to decode request.")
	}
	u := req_body.URL

	if u == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing URL parameter")
	}

	id := GenerateShortLink(u)
	t := time.Now()

	err = InsertURL(c.Request().Context(), URLEntry{
		ID:           id,
		Created_At:   t,
		Updated_At:   t,
		Redirect_URL: u,
	})

	if err != nil {
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Unable to shorten URL: %s", err.Error()))
	}

	return c.JSON(http.StatusOK, &successResponse{
		TS:  t.UnixNano(),
		URL: "http://localhost:8082/" + id,
	})
}

func GenerateShortLink(initialLink string) string {
	urlHashBytes := sha256Of(initialLink)
	generatedNumber := new(big.Int).SetBytes(urlHashBytes).Uint64()
	finalString := base58Encoded([]byte(fmt.Sprintf("%d", generatedNumber)))
	return finalString[:8]
}

func sha256Of(input string) []byte {
	algorithm := sha256.New()
	algorithm.Write([]byte(input))
	return algorithm.Sum(nil)
}

func base58Encoded(bytes []byte) string {
	encoding := base58.BitcoinEncoding
	encoded, _ := encoding.Encode(bytes)
	return string(encoded)
}
