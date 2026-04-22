// Package database provides MongoDB connection helpers.
package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Open creates a new MongoDB client, verifies connectivity with retries, and
// returns the client and the named database handle.
func Open(ctx context.Context, uri, dbName string) (*mongo.Client, *mongo.Database, error) {
	opts := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(25).
		SetMinPoolSize(10).
		SetMaxConnIdleTime(5 * time.Minute)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("connect mongo: %w", err)
	}

	var pingErr error
	for attempt := 1; attempt <= 20; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		pingErr = client.Ping(pingCtx, readpref.Primary())
		cancel()
		if pingErr == nil {
			return client, client.Database(dbName), nil
		}

		select {
		case <-ctx.Done():
			_ = client.Disconnect(context.Background())
			return nil, nil, fmt.Errorf("ping mongo: %w", ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}

	_ = client.Disconnect(context.Background())
	return nil, nil, fmt.Errorf("ping mongo after retries: %w", pingErr)
}
