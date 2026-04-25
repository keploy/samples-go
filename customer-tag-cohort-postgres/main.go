// customer-tag-cohort-postgres reproduces issue #4 — intra-bucket FIFO desync
// in cohort consumption when many recordings share the same SQL hash AND the
// same primary bucket bind value.
//
// Background. The keploy v3 mock store buckets recorded postgres frames by
// (sql_hash, primary_bind_value). When a single test window contains many
// calls to the same SELECT with the same first bind, those calls all land in
// the same bucket. Replay must consume them in record-time FIFO order.
//
// The bug. If the consumer mis-orders the bucket (e.g. iterates a Go map or a
// non-stable index), call N at replay can pick up the response intended for
// call M. Body diff catches it as a regression.
//
// The reproducer. Schema is customers(id) + tags(id, customer_id, tag,
// priority, created_at). Pre-seed 50 tags per customer for ids {11, 202, 203}
// — heterogeneous priorities, varying tag content per row. Endpoint
// GET /customers/:id/tags returns
//
//   SELECT id, customer_id, tag, priority FROM tags
//   WHERE customer_id = $1 ORDER BY id LIMIT $2 OFFSET $3
//
// The exerciser hits the same customer (id=11) five times in a single test
// window with offset=0,1,2,3,4 — same SQL hash, same primary bind, but five
// distinct rowsets. If the cohort consumer holds FIFO order, replay matches.
// If it desyncs, body diff fails on at least one of the five.
//
// Endpoints:
//   GET  /health              — readiness
//   GET  /customers/:id/tags  — the homogeneous-bucket SELECT
//   GET  /tag/:id             — single-row lookup (different SQL hash)
//   POST /tags                — insert (returns id; different SQL hash)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	http.HandleFunc("/customers/", handleCustomerRoute) // /customers/:id/tags
	http.HandleFunc("/tag/", handleTagByID)             // /tag/:id
	http.HandleFunc("/tags", handleTagsCreate)          // POST /tags

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Bring up the HTTP listener immediately so keploy's wait_for_http poll
	// passes before initDB has finished. /health flips to "ok" once ready.
	go func() {
		log.Printf("listening on :%s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("server failed on port %s: %v", port, err)
		}
	}()

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

// initDB creates the schema and pre-seeds three customers (id=11, 202, 203)
// with 50 tags each. Tag content varies per row so that two rows for the
// SAME customer differ — this matters because the bug only shows up when
// the homogeneous-bucket cohort has DIFFERENT response payloads per call.
func initDB() {
	ctx := context.Background()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS customers (
			id   INT PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			id          SERIAL PRIMARY KEY,
			customer_id INT NOT NULL REFERENCES customers(id),
			tag         TEXT NOT NULL,
			priority    INT NOT NULL DEFAULT 1
		)`,
		`INSERT INTO customers (id, name) VALUES
			(11,  'Acme Corp'),
			(202, 'Globex Industries'),
			(203, 'Initech Holdings')
		ON CONFLICT (id) DO NOTHING`,
		// Wipe existing tags so re-runs over a persisted volume don't pile
		// up duplicates (which would shift the LIMIT/OFFSET windows and
		// produce non-reproducible recordings).
		`TRUNCATE TABLE tags RESTART IDENTITY`,
	}
	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s); err != nil {
			log.Fatalf("initDB ddl: %v", err)
		}
	}

	// Pre-seed 50 tags per customer with varied tag content + priority. The
	// priority cycle (1,2,3,1,2,3,...) and tag-string interpolation make
	// every row distinct, so the SELECT response payload genuinely changes
	// row-by-row inside a single customer's bucket.
	for _, cid := range []int{11, 202, 203} {
		for i := 0; i < 50; i++ {
			tag := fmt.Sprintf("tag-c%d-r%02d-%s", cid, i, []string{"alpha", "beta", "gamma", "delta", "omega"}[i%5])
			priority := (i % 3) + 1
			if _, err := pool.Exec(ctx,
				`INSERT INTO tags (customer_id, tag, priority) VALUES ($1, $2, $3)`,
				cid, tag, priority); err != nil {
				log.Fatalf("initDB seed: %v", err)
			}
		}
	}
	log.Println("Database initialized: 3 customers, 150 tags (50 per customer)")
}

type Tag struct {
	ID         int    `json:"id"`
	CustomerID int    `json:"customer_id"`
	Tag        string `json:"tag"`
	Priority   int    `json:"priority"`
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	s := "starting"
	if ready {
		s = "ok"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": s})
}

// handleCustomerRoute dispatches /customers/:id/tags to the cohort SELECT.
// The exerciser hits this endpoint 5+ times for the SAME id within one test
// window — that's the homogeneous-bucket pattern. The query parameter
// `offset` is the knob that keeps the SQL hash + primary bind identical
// while VARYING the response rowset across calls.
func handleCustomerRoute(w http.ResponseWriter, r *http.Request) {
	if !ready {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	// Path shape: /customers/{id}/tags
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/customers/"), "/")
	if len(parts) != 2 || parts[1] != "tags" {
		http.Error(w, "expected /customers/:id/tags", http.StatusNotFound)
		return
	}
	customerID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "customer id must be int", http.StatusBadRequest)
		return
	}

	// LIMIT/OFFSET come from query params with safe defaults. The exerciser
	// drives offset=0,1,2,3,4 across five consecutive calls so the bound
	// SQL and the first bind ($1=customer_id) stay identical, but the
	// returned rowset shifts by one row each call.
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 500 {
			limit = n
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	// This is the homogeneous-bucket SQL — same prepared statement text on
	// every call from the exerciser, same $1=customer_id bind. Only $2
	// (limit) and $3 (offset) move, but those are NOT the primary bucket
	// key in keploy's cohort logic — the bucket is keyed by ($1, sql_hash).
	rows, err := getPool().Query(r.Context(),
		`SELECT id, customer_id, tag, priority FROM tags
		 WHERE customer_id = $1
		 ORDER BY id
		 LIMIT $2 OFFSET $3`,
		customerID, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("query: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	out := make([]Tag, 0, limit)
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.CustomerID, &t.Tag, &t.Priority); err != nil {
			http.Error(w, fmt.Sprintf("scan: %v", err), http.StatusInternalServerError)
			return
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("rows: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"customer_id": customerID,
		"limit":       limit,
		"offset":      offset,
		"count":       len(out),
		"tags":        out,
	})
}

// handleTagByID is a single-row SELECT — different SQL hash from the cohort
// query above. Included to verify replay still works for non-cohort traffic
// in the same test window.
func handleTagByID(w http.ResponseWriter, r *http.Request) {
	if !ready {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/tag/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "tag id must be int", http.StatusBadRequest)
		return
	}
	var t Tag
	err = getPool().QueryRow(r.Context(),
		`SELECT id, customer_id, tag, priority FROM tags WHERE id = $1`, id).
		Scan(&t.ID, &t.CustomerID, &t.Tag, &t.Priority)
	if err != nil {
		http.Error(w, fmt.Sprintf("tag %d: %v", id, err), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// handleTagsCreate inserts a tag and returns the new id. Same as above —
// distinct SQL hash from the cohort query, used for parity with the
// upstream sap_demo_java reproducer pattern.
func handleTagsCreate(w http.ResponseWriter, r *http.Request) {
	if !ready {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		CustomerID int    `json:"customer_id"`
		Tag        string `json:"tag"`
		Priority   int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, fmt.Sprintf("decode: %v", err), http.StatusBadRequest)
		return
	}
	if in.Priority == 0 {
		in.Priority = 1
	}
	var newID int
	err := getPool().QueryRow(r.Context(),
		`INSERT INTO tags (customer_id, tag, priority) VALUES ($1, $2, $3) RETURNING id`,
		in.CustomerID, in.Tag, in.Priority).Scan(&newID)
	if err != nil {
		http.Error(w, fmt.Sprintf("insert: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": newID})
}
