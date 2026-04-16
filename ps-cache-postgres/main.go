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
)

func main() {
	var err error
	// Retry connection for up to 30s to handle Docker compose startup ordering
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
	defer pool.Close()

	if os.Getenv("INIT_DB") == "true" {
		initDB()
	}

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/account", handleAccount)
	http.HandleFunc("/evict", handleEvict)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("server failed: %v", port, err)
	}
}

func initDB() {
	ctx := context.Background()
	stmts := []string{
		`CREATE SCHEMA IF NOT EXISTS travelcard`,
		`CREATE TABLE IF NOT EXISTS travelcard.travel_account (
			id SERIAL PRIMARY KEY, member_id INT NOT NULL UNIQUE,
			name TEXT NOT NULL, balance INT NOT NULL DEFAULT 0)`,
		`INSERT INTO travelcard.travel_account (member_id, name, balance) VALUES
			(19, 'Alice', 1000), (23, 'Bob', 2500),
			(31, 'Charlie', 500), (42, 'Diana', 7500)
		ON CONFLICT (member_id) DO NOTHING`,
	}
	for _, s := range stmts {
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
		if n, err := strconv.Atoi(v); err == nil {
			maxConns = n
		}
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = int32(maxConns)
	return pgxpool.NewWithConfig(ctx, cfg)
}

func getPool() *pgxpool.Pool {
	poolMu.RLock()
	defer poolMu.RUnlock()
	return pool
}

type Account struct {
	ID       int    `json:"id"`
	MemberID int    `json:"member_id"`
	Name     string `json:"name"`
	Balance  int    `json:"balance"`
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("health: encode error: %v", err)
	}
}

func handleAccount(w http.ResponseWriter, r *http.Request) {
	memberStr := r.URL.Query().Get("member")
	memberID, err := strconv.Atoi(memberStr)
	if err != nil {
		http.Error(w, "member param required (int)", 400)
		return
	}
	ctx := r.Context()
	p := getPool()
	tx, err := p.Begin(ctx)
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
	if err := json.NewEncoder(w).Encode(acct); err != nil {
		log.Printf("account: encode error: %v", err)
	}
}

func handleEvict(w http.ResponseWriter, r *http.Request) {
	newP, err := newPool(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	poolMu.Lock()
	oldPool := pool
	pool = newP
	poolMu.Unlock()
	oldPool.Close()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"evicted": "true"}); err != nil {
		log.Printf("evict: encode error: %v", err)
	}
}
