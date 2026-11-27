package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func InitPostgres(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS products (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		price DECIMAL(10,2) NOT NULL,
		quantity INTEGER NOT NULL,
		metadata JSONB,
		related_ids JSONB,
		categories JSONB,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS product_ratings (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		product_id UUID REFERENCES products(id) ON DELETE CASCADE,
		score DECIMAL(2,1) NOT NULL CHECK (score >= 1 AND score <= 5),
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS product_tags (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		product_id UUID REFERENCES products(id) ON DELETE CASCADE,
		tag VARCHAR(100) NOT NULL,
		created_at TIMESTAMP DEFAULT NOW(),
		UNIQUE(product_id, tag)
	);

	CREATE INDEX IF NOT EXISTS idx_product_tags_tag ON product_tags(tag);
	CREATE INDEX IF NOT EXISTS idx_product_ratings_product_id ON product_ratings(product_id);
	`

	_, err := db.Exec(query)
	return err
}