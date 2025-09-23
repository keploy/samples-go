// Package main initializes and starts a URL shortening service using Echo framework,
// connecting to a MySQL database and providing endpoints for shortening and resolving URLs
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hermione/echo-mysql/uss"
	"github.com/joho/godotenv"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	time.Sleep(2 * time.Second)
	appConfig, err := godotenv.Read()
	if err != nil {
		log.Printf("Error reading .env file %s", err.Error())
		os.Exit(1)

	}

	uss.MetaStore = &uss.Store{}
	err = uss.MetaStore.Connect(appConfig)
	if err != nil {
		log.Printf("Failed to connect to db %s", err.Error())
		os.Exit(1)
	}
	StartHTTPServer()
}

func StartHTTPServer() {
	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `${remote_ip} [${time_rfc3339}] "${method} ${uri} HTTP/1.0" ${status} ${latency_human} ${bytes_out} ${error} "${user_agent}"` + "\n",
		Skipper: func(c echo.Context) bool {
			return c.Request().RequestURI == "/healthcheck"
		},
	}))
	e.Use(middleware.Recover())
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.GET("/healthcheck", func(c echo.Context) error {
		return c.String(http.StatusOK, "good!")
	})

	e.GET("/resolve/:code", func(c echo.Context) error {
		code := c.Param("code")
		info := uss.MetaStore.FindByShortCode(code)
		if info != nil {
			return c.JSON(http.StatusOK, info)
		}

		return c.String(http.StatusNotFound, "Not Found.")
	})

	e.POST("/shorten", func(c echo.Context) error {
		req := new(uss.ShortCodeInfo)
		if err := c.Bind(req); err != nil {
			return err
		}

		req.ShortCode = uss.GenerateShortLink(req.URL)
		err := uss.MetaStore.Persist(req)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed Persisiting Entity with Error %s", err.Error()))
		}

		req.UpdatedAt = req.UpdatedAt.Truncate(time.Second)
		return c.JSON(http.StatusOK, req)
	})

	e.POST("/seed", func(c echo.Context) error {

		end := time.Date(9999, 1, 1, 0, 0, 0, 0, time.Local)

		// Populate the new CreatedBy field instead of Random
		info := &uss.ShortCodeInfo{
			EndTime:   end,
			CreatedBy: "dev@keploy.io",
		}

		if err := uss.MetaStore.Persist(info); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, info)
	})

	e.GET("/query/active", func(c echo.Context) error {
		// Call the new data store function that doesn't need any parameters.
		infos := uss.MetaStore.FindActive()

		// Return the found records as JSON.
		// If nothing is found, this will correctly return an empty JSON array: []
		return c.JSON(http.StatusOK, infos)
	})

	// automatically add routers for net/http/pprof e.g. /debug/pprof, /debug/pprof/heap, etc.
	// go get github.com/hiko1129/echo-pprof
	//echopprof.Wrap(e)
	e.Logger.Fatal(e.Start(":9090"))
}
