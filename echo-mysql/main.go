// Package main implements the HTTP server for the URL shortening service.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hermione/echo-mysql/uss"
	"github.com/joho/godotenv"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// ensure UTC in JSON responses for determinism across environments
func utcInfo(in *uss.ShortCodeInfo) *uss.ShortCodeInfo {
	if in == nil {
		return nil
	}
	cp := *in
	cp.EndTime = cp.EndTime.UTC()
	cp.UpdatedAt = cp.UpdatedAt.UTC()
	return &cp
}
func utcInfos(in []uss.ShortCodeInfo) []uss.ShortCodeInfo {
	out := make([]uss.ShortCodeInfo, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].EndTime = in[i].EndTime.UTC()
		out[i].UpdatedAt = in[i].UpdatedAt.UTC()
	}
	return out
}

func main() {
	time.Sleep(2 * time.Second)

	appConfig, err := godotenv.Read()
	if err != nil {
		log.Printf("Error reading .env file %s", err.Error())
		os.Exit(1)
	}

	uss.MetaStore = &uss.Store{}
	if err := uss.MetaStore.Connect(appConfig); err != nil {
		log.Printf("Failed to connect to db %s", err.Error())
		os.Exit(1)
	}

	// Build server (non-blocking)
	e := StartHTTPServer()

	// Start server in background goroutine
	go func() {
		if err := e.Start(":9090"); err != nil && err != http.ErrServerClosed {
			e.Logger.Errorf("server start failed: %v", err)
		}
	}()

	// Coordinated, graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done() // wait for signal

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Errorf("server shutdown error: %v", err)
	}

}

func StartHTTPServer() *echo.Echo {
	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format:  `${remote_ip} [${time_rfc3339}] "${method} ${uri} HTTP/1.0" ${status} ${latency_human} ${bytes_out} ${error} "${user_agent}"` + "\n",
		Skipper: func(c echo.Context) bool { return c.Request().RequestURI == "/healthcheck" },
	}))
	e.Use(middleware.Recover())

	e.GET("/", func(c echo.Context) error { return c.String(http.StatusOK, "Hello, World!") })
	e.GET("/healthcheck", func(c echo.Context) error { return c.String(http.StatusOK, "good!") })

	e.GET("/resolve/:code", func(c echo.Context) error {
		code := c.Param("code")
		info := uss.MetaStore.FindByShortCode(code)
		if info != nil {
			return c.JSON(http.StatusOK, utcInfo(info))
		}
		return c.String(http.StatusNotFound, "Not Found.")
	})

	e.POST("/shorten", func(c echo.Context) error {
		req := new(uss.ShortCodeInfo)
		if err := c.Bind(req); err != nil {
			return err
		}
		req.ShortCode = uss.GenerateShortLink(req.URL)
		if err := uss.MetaStore.Persist(req); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed Persisiting Entity with Error %s", err.Error()))
		}
		req.UpdatedAt = req.UpdatedAt.Truncate(time.Second)
		return c.JSON(http.StatusOK, utcInfo(req))
	})

	// Original "seed" kept (now sets CreatedBy too)
	e.POST("/seed", func(c echo.Context) error {
		end := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
		info := &uss.ShortCodeInfo{
			ShortCode: "dt-sentinel-9999-01-01T00:00:00Z",
			URL:       "https://example.com/sentinel-start",
			EndTime:   end,
			CreatedBy: "keploy.io/dates",
		}
		if err := uss.MetaStore.UpsertByShortCode(info); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, utcInfo(info))
	})

	// seed a set of edge-case datetimes
	e.POST("/seed/dates", func(c echo.Context) error {
		nowMicro := uss.ToMicroUTC(time.Now())
		payload := []*uss.ShortCodeInfo{
			{
				ShortCode: "dt-sentinel-9999-01-01T00:00:00Z",
				URL:       "https://example.com/sentinel-start",
				EndTime:   uss.SentinelStart,
				CreatedBy: "keploy.io/dates",
			},
			{
				ShortCode: "dt-max-9999-12-31T23:59:59.999999Z",
				URL:       "https://example.com/sentinel-max",
				EndTime:   uss.SentinelMax,
				CreatedBy: "keploy.io/dates",
			},
			{
				ShortCode: "dt-min-1000-01-01T00:00:00Z",
				URL:       "https://example.com/min-valid",
				EndTime:   time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC),
				CreatedBy: "keploy.io/dates",
			},
			{
				ShortCode: "dt-epoch-1970-01-01T00:00:00Z",
				URL:       "https://example.com/epoch",
				EndTime:   time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
				CreatedBy: "keploy.io/dates",
			},
			{
				ShortCode: "dt-leap-2020-02-29T12:34:56Z",
				URL:       "https://example.com/leap",
				EndTime:   time.Date(2020, 2, 29, 12, 34, 56, 0, time.UTC),
				CreatedBy: "keploy.io/dates",
			},
			{
				ShortCode: "dt-offset-2023-07-01T18:30:00+05:30",
				URL:       "https://example.com/offset",
				EndTime:   time.Date(2023, 7, 1, 18, 30, 0, 0, time.FixedZone("IST", 5*3600+30*60)),
				CreatedBy: "keploy.io/dates",
			},
			{
				ShortCode: "dt-now-trunc",
				URL:       "https://example.com/now",
				EndTime:   nowMicro,
				CreatedBy: "keploy.io/dates",
			},
		}

		if err := uss.MetaStore.UpsertMany(payload); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		resp := make([]uss.ShortCodeInfo, 0, len(payload))
		for _, p := range payload {
			if got := uss.MetaStore.FindByShortCode(p.ShortCode); got != nil {
				resp = append(resp, *utcInfo(got))
			}
		}
		return c.JSON(http.StatusOK, resp)
	})

	// exact EndTime query (RFC3339/RFC3339Nano/MySQL-like)
	e.GET("/query/by-endtime", func(c echo.Context) error {
		ts := c.QueryParam("ts")
		if ts == "" {
			return c.String(http.StatusBadRequest, "query param 'ts' required (RFC3339 or MySQL datetime)")
		}
		t, err := uss.ParseFlexible(ts)
		if err != nil {
			return c.String(http.StatusBadRequest, fmt.Sprintf("parse error: %v", err))
		}
		infos := uss.MetaStore.FindByEndTime(t)
		return c.JSON(http.StatusOK, utcInfos(infos))
	})

	// fetch both sentinels
	e.GET("/query/sentinels", func(c echo.Context) error {
		infos := uss.MetaStore.FindSentinels()
		return c.JSON(http.StatusOK, utcInfos(infos))
	})

	// fetch any seeded date rows
	e.GET("/query/dates", func(c echo.Context) error {
		infos := uss.MetaStore.FindSeededDates()
		return c.JSON(http.StatusOK, utcInfos(infos))
	})

	// fetch by label (stored in ShortCode)
	e.GET("/query/label/:label", func(c echo.Context) error {
		label := c.Param("label")
		info := uss.MetaStore.FindByShortCode(label)
		if info == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Not Found."})
		}
		return c.JSON(http.StatusOK, utcInfo(info))
	})

	return e
}
