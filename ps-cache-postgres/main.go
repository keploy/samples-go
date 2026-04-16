// ps-cache-postgres demonstrates and validates the PS-cache mock mismatch fix.
//
// It uses PgBouncer in transaction pooling mode (pool_size=1) so all HTTP
// requests share a single upstream PG connection.  The pgx driver caches
// prepared statements per connection, so the first request sends
// Parse(query="SELECT...") and subsequent requests send Bind-only.
//
// The /evict endpoint tells PgBouncer to cycle the upstream connection,
// creating a new connection with a cold PS cache.  This creates the
// recording-connection affinity scenario where mocks from different
// PgBouncer-level connections must be correctly matched during replay.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:6432/testdb?sslmode=disable"
	}
	maxConns := 1
	if v := os.Getenv("POOL_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxConns = n
		}
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("parse config: %v", err)
	}
	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = int32(maxConns)

	pool, err = pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/account", handleAccount)
	http.HandleFunc("/evict", handleEvict)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type Account struct {
	ID       int    `json:"id"`
	MemberID int    `json:"member_id"`
	Name     string `json:"name"`
	Balance  int    `json:"balance"`
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAccount does BEGIN → SELECT → COMMIT using the pooled connection.
// pgx caches the prepared statement after the first call.
func handleAccount(w http.ResponseWriter, r *http.Request) {
	memberStr := r.URL.Query().Get("member")
	memberID, err := strconv.Atoi(memberStr)
	if err != nil {
		http.Error(w, "member param required (int)", 400)
		return
	}
	ctx := r.Context()

	tx, err := pool.Begin(ctx)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer tx.Rollback(ctx)

	var acct Account
	err = tx.QueryRow(ctx,
		"SELECT id, member_id, name, balance FROM travelcard.travel_account WHERE member_id = $1",
		memberID).Scan(&acct.ID, &acct.MemberID, &acct.Name, &acct.Balance)
	if err != nil {
		http.Error(w, fmt.Sprintf("member %d: %v", memberID, err), 500)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(acct)
}

// handleEvict forces PgBouncer to cycle connections by closing and
// reconnecting the pool.  This simulates connection pool eviction
// in production (e.g., HikariCP eviction, PgBouncer server_idle_timeout).
func handleEvict(w http.ResponseWriter, r *http.Request) {
	// Close current pool and create a new one to force new PG connection.
	oldPool := pool

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:6432/testdb?sslmode=disable"
	}
	maxConns := 1
	if v := os.Getenv("POOL_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxConns = n
		}
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = int32(maxConns)

	newPool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	pool = newPool
	oldPool.Close()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"evicted": "true"})
}
