// Package main is a minimal HTTP server used by the Keploy sampling-test
// pipeline. The /work handler sleeps for HANDLER_DELAY (default 500ms) so a
// burst of concurrent requests overlaps inside the Keploy proxy, exercising
// the --enable-sampling slot semaphore in pkg/agent/proxy/incoming.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

var counter atomic.Uint64

func parseDelay() time.Duration {
	if v := os.Getenv("HANDLER_DELAY_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms >= 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return 500 * time.Millisecond
}

func main() {
	delay := parseDelay()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/work", func(w http.ResponseWriter, r *http.Request) {
		id := counter.Add(1)
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         id,
			"path":       r.URL.Path,
			"delayMs":    delay.Milliseconds(),
			"clientAddr": r.RemoteAddr,
		})
	})

	addr := ":" + port
	log.Printf("sampling-test listening on %s (handler delay %s)", addr, delay)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
