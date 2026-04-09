package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// ---------------------------------------------------------------------------
// Config from environment
// ---------------------------------------------------------------------------

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

var (
	listenAddr         = env("LISTEN_ADDR", ":8080")
	dbDSN              = env("DATABASE_URL", "postgres://repro:repro@localhost:5432/reprodb?sslmode=disable")
	httpsTarget        = env("HTTPS_TARGET", "https://httpbin.org/get")
	httpProxyURL       = env("HTTP_PROXY_URL", "")        // e.g. http://forward-proxy:3128
	concurrentConns    = envInt("CONCURRENT_CONNS", 20)    // number of parallel HTTPS connections per request
	otelEndpoint       = env("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318")
	otelEnabled        = env("OTEL_ENABLED", "true") == "true"
	otelInterval       = envDuration("OTEL_EXPORT_INTERVAL", 5*time.Second)  // configurable OTel export interval
	bgNoiseConns       = envInt("BG_NOISE_CONNS", 0)                         // background HTTP noise connections per second
)

// ---------------------------------------------------------------------------
// OpenTelemetry setup — exports to localhost:4318 (no collector)
//
// Issue 2 reproduction: When Keploy proxy intercepts this traffic during
// recording, it gets "connection refused". During replay, mock-not-found
// errors fill the 100-buffer error channel and block test coordination.
// ---------------------------------------------------------------------------

func initOTel(ctx context.Context) (func(), error) {
	if !otelEnabled {
		return func() {}, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("repro-app"),
			semconv.ServiceVersion("0.1.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	// Trace exporter — OTLP HTTP to localhost:4318
	traceExp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otelEndpoint),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: 1 * time.Second,
			MaxInterval:     5 * time.Second,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("creating trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp, sdktrace.WithBatchTimeout(otelInterval)),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Metric exporter — OTLP HTTP to localhost:4318
	metricExp, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(otelEndpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating metric exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExp, metric.WithInterval(otelInterval))),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
	}
	return shutdown, nil
}

// ---------------------------------------------------------------------------
// Database setup — large rows to trigger Postgres decode bug
//
// Issue 3 reproduction: DataRow responses > 1KB split across TCP segments.
// The Keploy Postgres parser tries to decode before all segments arrive,
// causing "incomplete or invalid response packet (DataRow)" errors.
// ---------------------------------------------------------------------------

func initDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping failed: %w", err)
	}
	return db, nil
}

// ---------------------------------------------------------------------------
// HTTPS client — one transport per call = one TLS handshake = one cert gen
//
// Issue 1 reproduction: Each newTransport() creates a fresh TCP connection
// through the forward proxy (HTTP CONNECT tunnel). Keploy MITM intercepts
// each and generates a NEW certificate for the same hostname. With
// concurrentConns=20, that's 20 cert generations in parallel — no caching.
//
// Issue 4 reproduction: The HTTP traffic through the CONNECT tunnel is first
// checked by the SQS parser (which matches any "POST " prefix) before
// falling back to the HTTP parser.
// ---------------------------------------------------------------------------

func newTransport() *http.Transport {
	t := &http.Transport{
		// Force new connections — no pooling, no keep-alive.
		// Each request = new TCP conn = new CONNECT tunnel = new cert.
		DisableKeepAlives: true,
		MaxIdleConns:      0,
		IdleConnTimeout:   1 * time.Nanosecond,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // intentional: accept Keploy MITM proxy cert
		},
		// Custom dialer with short timeouts
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: -1, // disable TCP keep-alive
		}).DialContext,
	}

	// If a forward proxy is configured, route through it (HTTP CONNECT tunnel)
	if httpProxyURL != "" {
		proxyURL, err := url.Parse(httpProxyURL)
		if err == nil {
			t.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return t
}

func fetchHTTPS(ctx context.Context, targetURL string) (int, string, error) {
	client := &http.Client{
		Transport: newTransport(),
		Timeout:   30 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return 0, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

type transferResult struct {
	HTTPSURL       string `json:"https_url"`
	HTTPSStatus    int    `json:"https_status,omitempty"`
	HTTPSError     string `json:"https_error,omitempty"`
	DBLargeRows    int    `json:"db_large_rows"`
	DBError        string `json:"db_error,omitempty"`
	ConcurrentReqs int    `json:"concurrent_requests"`
	Duration       string `json:"duration"`
}

// /api/transfer — the main endpoint that triggers all issues simultaneously
func transferHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()
		result := transferResult{
			HTTPSURL:       httpsTarget,
			ConcurrentReqs: concurrentConns,
		}

		// ----- Issue 1: Concurrent HTTPS through proxy (cert storm) -----
		type httpsResult struct {
			status int
			err    error
		}
		results := make([]httpsResult, concurrentConns)
		var wg sync.WaitGroup
		for i := 0; i < concurrentConns; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				status, _, err := fetchHTTPS(ctx, httpsTarget)
				results[idx] = httpsResult{status: status, err: err}
			}(i)
		}
		wg.Wait()

		// Report first success or last error
		for _, r := range results {
			if r.err != nil {
				result.HTTPSError = r.err.Error()
			} else {
				result.HTTPSStatus = r.status
				result.HTTPSError = ""
				break
			}
		}

		// ----- Issue 3: Postgres large rows (100KB each = 5MB+ response) -----
		// Single query returns 50 rows × 100KB = 5MB — each DataRow packet
		// spans ~70 TCP segments, forcing the parser to buffer across segments.
		rows, err := db.QueryContext(ctx,
			`SELECT id, name, description, large_payload FROM large_records ORDER BY id LIMIT 50`)
		if err != nil {
			result.DBError = err.Error()
		} else {
			count := 0
			for rows.Next() {
				var id int
				var name, desc, payload string
				if err := rows.Scan(&id, &name, &desc, &payload); err != nil {
					result.DBError = err.Error()
					break
				}
				count++
			}
			if rowErr := rows.Err(); rowErr != nil {
				result.DBError = rowErr.Error()
			}
			rows.Close()
			result.DBLargeRows = count
		}

		// Also query wide_records (20 columns × 3.2KB each = 64KB per DataRow)
		wideRows, err := db.QueryContext(ctx,
			`SELECT * FROM wide_records LIMIT 20`)
		if err == nil {
			for wideRows.Next() {
				var id int
				cols := make([]string, 20)
				ptrs := make([]interface{}, 21)
				ptrs[0] = &id
				for i := range cols {
					ptrs[i+1] = &cols[i]
				}
				if scanErr := wideRows.Scan(ptrs...); scanErr != nil {
				break
			}
			}
			wideRows.Close()
		}

		result.Duration = time.Since(start).String()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// /api/batch-transfer — fires even more concurrent HTTPS connections
func batchTransferHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()
		batchSize := envInt("BATCH_SIZE", 40) // match the 42 connections from production

		type batchResult struct {
			TotalRequests int      `json:"total_requests"`
			Succeeded     int      `json:"succeeded"`
			Failed        int      `json:"failed"`
			Errors        []string `json:"errors,omitempty"`
			DBRows        int      `json:"db_rows"`
			Duration      string   `json:"duration"`
		}

		var (
			mu        sync.Mutex
			succeeded int
			failed    int
			errors    []string
		)

		// Fire batchSize concurrent HTTPS requests — each through a new connection
		var wg sync.WaitGroup
		for i := 0; i < batchSize; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				status, _, err := fetchHTTPS(ctx, httpsTarget)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					failed++
					if len(errors) < 5 {
						errors = append(errors, err.Error())
					}
				} else if status >= 200 && status < 300 {
					succeeded++
				} else {
					failed++
					errors = append(errors, fmt.Sprintf("HTTP %d", status))
				}
			}()
		}
		wg.Wait()

		// Also query multiple large Postgres result sets concurrently
		dbRows := 0
		for i := 0; i < 3; i++ {
			rows, err := db.QueryContext(ctx,
				`SELECT id, name, description, large_payload FROM large_records ORDER BY id LIMIT 100`)
			if err != nil {
				continue
			}
			for rows.Next() {
				var id int
				var name, desc, payload string
				_ = rows.Scan(&id, &name, &desc, &payload)
				dbRows++
			}
			rows.Close()
		}

		res := batchResult{
			TotalRequests: batchSize,
			Succeeded:     succeeded,
			Failed:        failed,
			Errors:        errors,
			DBRows:        dbRows,
			Duration:      time.Since(start).String(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

// /health — standard health check
func healthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		if err := db.PingContext(r.Context()); err != nil {
			status = "db_down: " + err.Error()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": status})
	}
}

// /api/post-transfer — POST endpoint to trigger SQS parser misclassification (Issue 4)
// The SQS parser matches any buffer starting with "POST " — this HTTP POST through
// the CONNECT tunnel will be first evaluated by the SQS parser before falling back.
func postTransferHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()

		// Make a POST request through the proxy — this triggers SQS parser match
		client := &http.Client{
			Transport: newTransport(),
			Timeout:   30 * time.Second,
		}
		postTarget := env("HTTPS_POST_TARGET", "https://httpbin.org/post")
		req, err := http.NewRequestWithContext(ctx, "POST", postTarget, bytes.NewReader([]byte(`{"action":"transfer","amount":100}`)))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		var status int
		var respErr string
		resp, err := client.Do(req)
		if err != nil {
			respErr = err.Error()
		} else {
			status = resp.StatusCode
			resp.Body.Close()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"method":   "POST",
			"target":   postTarget,
			"status":   status,
			"error":    respErr,
			"duration": time.Since(start).String(),
		})
	}
}

// backgroundNoise generates continuous outgoing HTTP connections to fill the
// Keploy proxy's error channel during replay. Each connection that doesn't
// match a recorded mock pushes an error to the 100-buffer errChannel.
// Combined with OTel exports, this saturates the channel quickly.
func backgroundNoise(ctx context.Context) {
	if bgNoiseConns <= 0 {
		return
	}
	log.Printf("Starting background noise: %d outgoing connections/sec", bgNoiseConns)
	ticker := time.NewTicker(time.Second / time.Duration(bgNoiseConns))
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			go func() {
				// Each of these during replay = "no matching mock found" = error to errChannel
				client := &http.Client{Transport: newTransport(), Timeout: 5 * time.Second}
				req, err := http.NewRequest("GET", httpsTarget, nil)
				if err != nil {
					return
				}
				resp, err := client.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}()
		}
	}
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init OTel (Issue 2 — exports to localhost:4318 with no collector)
	shutdownOTel, err := initOTel(ctx)
	if err != nil {
		log.Printf("OTel init returned error (expected if no collector): %v", err)
	} else {
		defer shutdownOTel()
	}

	// Init Postgres
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to connect to database (check DATABASE_URL env var): %v", err)
	}
	defer db.Close()

	// Start background noise (fills error channel during replay)
	go backgroundNoise(ctx)

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(db))
	mux.HandleFunc("/api/transfer", transferHandler(db))
	mux.HandleFunc("/api/batch-transfer", batchTransferHandler(db))
	mux.HandleFunc("/api/post-transfer", postTransferHandler(db))

	log.Printf("repro-app listening on %s", listenAddr)
	log.Printf("  HTTPS target: %s", httpsTarget)
	log.Printf("  HTTP proxy:   %s", httpProxyURL)
	log.Printf("  OTel endpoint: %s (enabled=%v, interval=%s)", otelEndpoint, otelEnabled, otelInterval)
	log.Printf("  DB: [redacted]")
	log.Printf("  Concurrent conns: %d", concurrentConns)
	log.Printf("  Background noise: %d conns/sec", bgNoiseConns)

	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatal(err)
	}
}
