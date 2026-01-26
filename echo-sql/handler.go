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
	"go.uber.org/zap"
)

type ConnectionDetails struct {
	host     string // The hostname (default "localhost")
	port     string // The port to connect on (default "5438")
	user     string // The username (default "postgres")
	password string // The password (default "postgres")
	dbName   string // The database name (default "postgres")
}

type URLEntry struct {
	ID          string
	RedirectURL string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type urlRequestBody struct {
	URL string `json:"url"`
}

type successResponse struct {
	TS  int64  `json:"ts"`
	URL string `json:"url"`
}

/*
NewConnection Establishes a connection with the PostgreSQL instance.
*/
func NewConnection(ctx context.Context, connDetails ConnectionDetails) (*sql.DB, error) {
	dbInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		connDetails.host, connDetails.port, connDetails.user, connDetails.password, connDetails.dbName)

	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		return nil, err
	}

	// Ping the database to ensure a connection can be established
	Logger.Info("Pinging database...")
	err = db.PingContext(ctx)
	if err != nil {
		Logger.Error("Ping failed", zap.Error(err))
		return nil, err
	}
	Logger.Info("Ping successful")

	// Check if the "url_map" table exists, create it if not
	Logger.Info("Creating table if not exists...")
	if err := createURLMapTable(ctx, db); err != nil {
		Logger.Error("Failed to create table", zap.Error(err))
		return nil, err
	}
	Logger.Info("Table creation check passed")

	return db, nil
}

func createURLMapTable(ctx context.Context, db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS url_map (
			id           VARCHAR(255) PRIMARY KEY,
			redirect_url VARCHAR(255) NOT NULL,
			created_at   TIMESTAMP NOT NULL,
			updated_at   TIMESTAMP NOT NULL
		);
	`

	_, err := db.ExecContext(ctx, query)
	if err != nil {
		Logger.Error("ExecContext failed for create table", zap.Error(err))
	}
	return err
}

func InsertURL(c context.Context, entry URLEntry) error {
	insertQuery := `
		INSERT INTO url_map (id, redirect_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`

	selectQuery := `
			SELECT *
			FROM url_map
			WHERE id = $1
	`

	updateTimestamp := `
			UPDATE url_map
			SET updated_at = $1
			WHERE id = $2
	`

	res, err := Database.QueryContext(c, selectQuery, entry.ID) // See if the URL already exists
	if err != nil {
		return err
	}
	defer handleDeferError(res.Close)
	if res.Next() { // If we get rows back, that means we have a duplicate URL
		var savedEntry URLEntry
		err := res.Scan(&savedEntry.ID, &savedEntry.RedirectURL, &savedEntry.CreatedAt, &savedEntry.UpdatedAt)
		if err != nil {
			return err
		}

		_, err = Database.ExecContext(c, updateTimestamp, entry.UpdatedAt, entry.ID)
		if err != nil {
			return err
		}

		return nil
	}

	_, err = Database.ExecContext(c, insertQuery, entry.ID, entry.RedirectURL, entry.CreatedAt, entry.UpdatedAt)
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

	findURLQuery := `
		SELECT *
		FROM url_map
		WHERE id = $1
	`
	res, err := Database.QueryContext(c.Request().Context(), findURLQuery, id) // Attempt to find a matching URL

	if err != nil {
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "Error encountered while attempting to lookup URL.")
	}
	defer handleDeferError(res.Close)
	if !res.Next() {
		return echo.NewHTTPError(http.StatusNotFound, "Invalid URL ID.")
	}

	var entry URLEntry
	err = res.Scan(&entry.ID, &entry.RedirectURL, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		Logger.Fatal(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not successfully scan retrieved DB entry.")
	}

	err = c.Redirect(http.StatusPermanentRedirect, entry.RedirectURL)
	if err != nil {
		return err
	}
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

	reqBody := new(urlRequestBody)
	err = c.Bind(reqBody)
	if err != nil {
		fmt.Println(reqBody)
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to decode request.")
	}

	u := reqBody.URL

	if u == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing URL parameter")
	}

	selectQuery := `
			SELECT *
			FROM url_map
			WHERE id = $1
	`

	updateTimestamp := `
			UPDATE url_map
			SET updated_at = $1, redirect_url = $2
			WHERE id = $3
	`

	res, err := Database.QueryContext(c.Request().Context(), selectQuery, id) // See if the URL already exists
	if err != nil {
		return err
	}
	defer handleDeferError(res.Close)
	if res.Next() { // If we get rows back, that means we have a duplicate URL
		var savedEntry URLEntry
		err := res.Scan(&savedEntry.ID, &savedEntry.RedirectURL, &savedEntry.CreatedAt, &savedEntry.UpdatedAt)
		if err != nil {
			return err
		}
		result, err := Database.ExecContext(c.Request().Context(), updateTimestamp, time.Now(), u, id)
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

	reqBody := new(urlRequestBody)

	err = c.Bind(reqBody)
	if err != nil {
		fmt.Println(reqBody)
		Logger.Error(err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to decode request.")
	}
	u := reqBody.URL

	if u == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing URL parameter")
	}

	id := GenerateShortLink(u)
	t := time.Now()

	err = InsertURL(c.Request().Context(), URLEntry{
		ID:          id,
		CreatedAt:   t,
		UpdatedAt:   t,
		RedirectURL: u,
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
