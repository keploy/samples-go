// Package database provides PostgreSQL connection helpers.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for database/sql
)

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	var pingErr error
	for attempt := 1; attempt <= 20; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		pingErr = db.PingContext(pingCtx)
		cancel()
		if pingErr == nil {
			return db, nil
		}

		select {
		case <-ctx.Done():
			_ = db.Close()
			return nil, fmt.Errorf("ping postgres: %w", ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}

	_ = db.Close()
	return nil, fmt.Errorf("ping postgres after retries: %w", pingErr)
}
