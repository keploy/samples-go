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
	// No SetMinPoolSize: keep the pool lazy so mongo connections open on the
	// first query rather than eagerly at startup. An eager pool opens its
	// connections before keploy's k8s sidecar agent attaches, so that traffic
	// is never intercepted and the mocks are missed; lazy pooling lets every
	// connection be recorded.
	opts := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(25).
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
			if dErr := client.Disconnect(context.Background()); dErr != nil {
				return nil, nil, fmt.Errorf("ping mongo: context done, disconnect: %v: %w", dErr, ctx.Err())
			}
			return nil, nil, fmt.Errorf("ping mongo: %w", ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}

	if dErr := client.Disconnect(context.Background()); dErr != nil {
		return nil, nil, fmt.Errorf("ping mongo after retries (disconnect: %v): %w", dErr, pingErr)
	}
	return nil, nil, fmt.Errorf("ping mongo after retries: %w", pingErr)
}
