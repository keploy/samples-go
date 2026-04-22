// Package database provides MySQL connection and schema helpers.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // register mysql driver
)

// Open creates a *sql.DB, verifies connectivity with retries, and applies the
// runtime schema. It returns the open DB handle; the caller must call db.Close().
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	// Retry loop — MySQL can take a few seconds to become ready.
	const maxAttempts = 20
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if pingErr := db.PingContext(ctx); pingErr == nil {
			break
		} else if attempt == maxAttempts {
			db.Close()
			return nil, fmt.Errorf("mysql did not become ready after %d attempts: %w", maxAttempts, pingErr)
		}
		select {
		case <-ctx.Done():
			db.Close()
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	return db, nil
}

// EnsureRuntimeSchema creates all tables and indexes if they do not already exist.
func EnsureRuntimeSchema(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS customers (
			id           CHAR(36)     NOT NULL PRIMARY KEY,
			email        VARCHAR(320) NOT NULL,
			full_name    VARCHAR(255) NOT NULL,
			segment      VARCHAR(64)  NOT NULL,
			created_at   DATETIME(3)  NOT NULL,
			UNIQUE KEY uq_customers_email (email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS products (
			id              CHAR(36)     NOT NULL PRIMARY KEY,
			sku             VARCHAR(128) NOT NULL,
			name            VARCHAR(255) NOT NULL,
			category        VARCHAR(128) NOT NULL,
			price_cents     INT          NOT NULL,
			inventory_count INT          NOT NULL DEFAULT 0,
			created_at      DATETIME(3)  NOT NULL,
			UNIQUE KEY uq_products_sku (sku)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS orders (
			id              CHAR(36)    NOT NULL PRIMARY KEY,
			customer_id     CHAR(36)    NOT NULL,
			customer_email  VARCHAR(320) NOT NULL,
			customer_name   VARCHAR(255) NOT NULL,
			customer_segment VARCHAR(64) NOT NULL,
			status          VARCHAR(32) NOT NULL,
			total_cents     INT         NOT NULL DEFAULT 0,
			created_at      DATETIME(3) NOT NULL,
			KEY idx_orders_customer_created (customer_id, created_at),
			KEY idx_orders_status_created   (status, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS order_items (
			id              CHAR(36)     NOT NULL PRIMARY KEY,
			order_id        CHAR(36)     NOT NULL,
			product_id      CHAR(36)     NOT NULL,
			sku             VARCHAR(128) NOT NULL,
			name            VARCHAR(255) NOT NULL,
			category        VARCHAR(128) NOT NULL,
			quantity        INT          NOT NULL,
			unit_price_cents INT         NOT NULL,
			line_total_cents INT         NOT NULL,
			KEY idx_order_items_order (order_id),
			KEY idx_order_items_product (product_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS large_payloads (
			id                 CHAR(36)      NOT NULL PRIMARY KEY,
			name               VARCHAR(255)  NOT NULL,
			content_type       VARCHAR(128)  NOT NULL,
			payload            LONGTEXT      NOT NULL,
			payload_size_bytes INT           NOT NULL,
			sha256             CHAR(64)      NOT NULL,
			created_at         DATETIME(3)   NOT NULL,
			KEY idx_large_payloads_created (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	}

	return nil
}
