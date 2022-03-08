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
	"github.com/keploy/go-sdk/integrations/ksql"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
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
func NewConnection(conn_details ConnectionDetails) (*sql.DB, error) {
	// Connect to PostgreSQL database
	db_info := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		conn_details.host, conn_details.port, conn_details.user, conn_details.password, conn_details.db_name)

	driver := ksql.Driver{Driver: pq.Driver{}}

	sql.Register("keploy", &driver)

	db, err := sql.Open("keploy", db_info)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func InsertURL(c context.Context, entry URLEntry) error {
	sql_query := `
		INSERT INTO url_map (id, redirect_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := Database.ExecContext(c, sql_query, entry.ID, entry.Redirect_URL, entry.Created_At, entry.Updated_At)
	if err != nil {
		fmt.Printf(err.Error())
		return err
	}

	return nil
}

func PutURL(c echo.Context) error {
	req_body := new(urlRequestBody)

	err := c.Bind(req_body)
	if err != nil {
		fmt.Println(req_body)
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
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to shorten URL", err)
	}

	return c.JSON(http.StatusOK, &successResponse{
		TS:  time.Now().UnixNano(),
		URL: "http://localhost:8080/" + id,
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
