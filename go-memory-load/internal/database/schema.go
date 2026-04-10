package database

import (
	"context"
	"database/sql"
	"fmt"
)

func EnsureRuntimeSchema(ctx context.Context, db *sql.DB) error {
	const query = `
		CREATE EXTENSION IF NOT EXISTS pgcrypto;

		CREATE TABLE IF NOT EXISTS large_payloads (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			content_type TEXT NOT NULL,
			payload_text TEXT NOT NULL,
			payload_size_bytes INTEGER NOT NULL CHECK (payload_size_bytes > 0),
			sha256 TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_large_payloads_created_at
			ON large_payloads (created_at DESC);

		CREATE INDEX IF NOT EXISTS idx_large_payloads_payload_size_bytes
			ON large_payloads (payload_size_bytes DESC);
	`

	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("ensure runtime schema: %w", err)
	}

	return nil
}
