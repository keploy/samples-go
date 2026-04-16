package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pool   *pgxpool.Pool
	poolMu sync.RWMutex
	ready  bool
)

func main() {
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/account", handleAccount)
	http.HandleFunc("/evict", handleEvict)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start HTTP server immediately so keploy health checks pass
	go func() {
		log.Printf("listening on :%s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("server failed on port %s: %v", port, err)
		}
	}()

	// Connect to DB with retries
	var err error
	for i := 0; i < 30; i++ {
		pool, err = newPool(context.Background())
		if err == nil {
			break
		}
		log.Printf("waiting for database (%d/30): %v", i+1, err)
		time.Sleep(time.Second)
	}
	if err != nil {
		log.Fatalf("failed to create pool after 30s: %v", err)
	}
	if os.Getenv("INIT_DB") == "true" {
		initDB()
	}
	ready = true
	log.Println("Application ready")
	select {}
}

func initDB() {
	ctx := context.Background()
	for _, s := range []string{
		`CREATE SCHEMA IF NOT EXISTS travelcard`,
		`CREATE TABLE IF NOT EXISTS travelcard.travel_account (
			id SERIAL PRIMARY KEY, member_id INT NOT NULL UNIQUE,
			name TEXT NOT NULL, balance INT NOT NULL DEFAULT 0)`,
		`INSERT INTO travelcard.travel_account (member_id, name, balance) VALUES
			(19, 'Alice', 1000), (23, 'Bob', 2500),
			(31, 'Charlie', 500), (42, 'Diana', 7500)
		ON CONFLICT (member_id) DO NOTHING`,
	} {
		if _, err := pool.Exec(ctx, s); err != nil {
			log.Fatalf("initDB: %v", err)
		}
	}
	log.Println("Database initialized")
}

func newPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/testdb?sslmode=disable"
	}
	maxConns := 1
	if v := os.Getenv("POOL_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil { maxConns = n }
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = int32(maxConns)
	return pgxpool.NewWithConfig(ctx, cfg)
}

func getPool() *pgxpool.Pool { poolMu.RLock(); defer poolMu.RUnlock(); return pool }

type Account struct {
	ID       int    `json:"id"`
	MemberID int    `json:"member_id"`
	Name     string `json:"name"`
	Balance  int    `json:"balance"`
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s := "starting"
	if ready { s = "ok" }
	json.NewEncoder(w).Encode(map[string]string{"status": s})
}

func handleAccount(w http.ResponseWriter, r *http.Request) {
	if !ready { http.Error(w, "not ready", 503); return }
	memberID, err := strconv.Atoi(r.URL.Query().Get("member"))
	if err != nil { http.Error(w, "member param required (int)", 400); return }
	p := getPool()
	tx, err := p.Begin(r.Context())
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer tx.Rollback(r.Context())
	var a Account
	err = tx.QueryRow(r.Context(),
		"SELECT id, member_id, name, balance FROM travelcard.travel_account WHERE member_id = $1",
		memberID).Scan(&a.ID, &a.MemberID, &a.Name, &a.Balance)
	if err != nil { http.Error(w, fmt.Sprintf("member %d: %v", memberID, err), 500); return }
	tx.Commit(r.Context())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func handleEvict(w http.ResponseWriter, r *http.Request) {
	newP, err := newPool(r.Context())
	if err != nil { http.Error(w, err.Error(), 500); return }
	poolMu.Lock(); oldPool := pool; pool = newP; poolMu.Unlock()
	oldPool.Close()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"evicted": "true"})
}
