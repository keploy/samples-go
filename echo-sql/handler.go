package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/itchyny/base58-go"
	"github.com/labstack/echo/v4"
)

type ConnectionDetails struct {
	host     string // The hostname (default "localhost")
	port     string // The port to connect on (default "5438")
	user     string // The username (default "postgres")
	password string // The password (default "postgres")
	db_name  string // The database name (default "postgres")
}

type errorResponse struct {
	err string `json:"error"`
}

type successResponse struct {
	ts  int64  `json:"ts"`
	url string `json:"url"`
}

/*
	Establishes a connection with the PostgreSQL instance.
*/
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

func PutURL(c echo.Context) error {
	var req_body map[string]string

	err := c.Bind(req_body)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{err: "Failed to decode request."})
		return err
	}
	u := req_body["url"]

	if u == "" {
		c.JSON(http.StatusBadRequest, errorResponse{err: "Missing url parameter"})
		return err
	}

	//t := time.Now()
	id := GenerateShortLink(u)
	// Insert into PostgreSQL database. (eventually)
	/*
		err = Upsert(c.Request.Context(), url{
			ID:      id,
			Created: t,
			Updated: t,
			URL:     u,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{err: err})
			return
		}
	*/
	fmt.Print("Hello, world!")
	c.JSON(http.StatusOK, successResponse{
		ts:  time.Now().UnixNano(),
		url: "http://localhost:8080/" + id,
	})
	return nil
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
