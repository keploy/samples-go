// Sample Aerospike-Go application that exercises the parser over a
// TLS-protected wire. It mirrors e2e-run/main.go endpoint-for-endpoint
// but talks to Aerospike on port 3001 with a *tls.Conn underneath.
//
// The point of this binary is to confirm two claims about the parser:
//
//  1. The Aerospike parser is TLS-blind: it imports no crypto/tls and
//     does not branch on PortClear/PortTLS/PortXDR. With this app
//     pointed at port 3001 over TLS, the parser still receives plain
//     Aerospike wire bytes — because Keploy's proxy terminates TLS
//     upstream and hands the parser a transport that already speaks
//     decrypted bytes.
//
//  2. Proxy TLS detection is byte-pattern driven, not port driven.
//     This client sends a real TLS ClientHello on a non-3306, non-443
//     port; the proxy still recognises it via Peek(5) → IsTLSHandshake
//     and MITMs the connection. Port 3001 is incidental.
//
// Endpoints (identical contract to e2e-run):
//
//	GET  /health           — info "build" + "namespaces"
//	POST /put              — single-record PUT
//	GET  /get/{key}        — single-record GET
//	POST /batch/put        — BATCH_WRITE
//	GET  /batch/get        — BATCH_READ
//	POST /scan             — full namespace scan
//	POST /query            — secondary-index range query
//	POST /udf              — UDF_EXECUTE
//	POST /cdt/list/append  — CDT list append
//	POST /cdt/map/put      — CDT map put
//	POST /touch/{key}      — TOUCH
//	DELETE /key/{key}      — DELETE
package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	as "github.com/aerospike/aerospike-client-go/v7"
	aslog "github.com/aerospike/aerospike-client-go/v7/logger"
)

func main() {
	host := flag.String("aerospike-host", env("AEROSPIKE_HOST", "127.0.0.1"), "aerospike server host")
	port := flag.Int("aerospike-port", envInt("AEROSPIKE_PORT", 3001), "aerospike server port (TLS service)")
	tlsName := flag.String("tls-name", env("AEROSPIKE_TLS_NAME", "aerospike.local"), "TLS hostname expected on the server certificate")
	caFile := flag.String("tls-ca", env("AEROSPIKE_TLS_CA", "certs/ca.pem"), "PEM file containing CA cert(s) that signed the server cert")
	certFile := flag.String("tls-cert", env("AEROSPIKE_TLS_CERT", ""), "optional client cert PEM (mutual TLS)")
	keyFile := flag.String("tls-key", env("AEROSPIKE_TLS_KEY", ""), "optional client key PEM (mutual TLS)")
	insecure := flag.Bool("tls-insecure", envBool("AEROSPIKE_TLS_INSECURE", false), "skip server cert verification (debug only)")
	listen := flag.String("listen", env("LISTEN", ":8080"), "http listen address")
	flag.Parse()

	aslog.Logger.SetLevel(aslog.DEBUG)

	tlsCfg, err := buildTLSConfig(*caFile, *certFile, *keyFile, *tlsName, *insecure)
	if err != nil {
		log.Fatalf("build tls config: %v", err)
	}

	policy := as.NewClientPolicy()
	policy.TlsConfig = tlsCfg
	// Aerospike CE only advertises clear-text peer addresses
	// (peers-clear-std); behind a TLS terminator like stunnel those
	// addresses are unreachable. Pin the client to the seed so it
	// doesn't try to open clear-text connections to peers.
	policy.SeedOnlyCluster = true
	// Pool sizing for the parallel handler: a single /parallel?n=N
	// curl fans out N goroutines, each grabbing its own pooled
	// connection. The default queue size (256) is fine, but the open
	// path is the real bottleneck under TLS-MITM replay — limit how
	// many fresh TLS dials race the pool's connect-or-wait latch by
	// pinning a generous concurrent-open budget. ConnectionQueueSize
	// is set explicitly so it survives a future default-tuning drift.
	policy.ConnectionQueueSize = 256
	// Hold OpeningConnectionThreshold low: stunnel's fork model on
	// the docker side forks one child per accepted connection, and
	// fork() is slow enough that a burst of 64+ concurrent dials
	// can outpace the kernel's accept-queue + stunnel's fork rate
	// at record time, producing EOFs on the proxy's upstream dial.
	// 16 is a comfortable number for stunnel and big enough that
	// /parallel?n=24 still fans out usefully.
	policy.OpeningConnectionThreshold = 16
	// Without TLSName on the Host, the client will not negotiate TLS
	// for this host even with TlsConfig set.
	h := as.NewHost(*host, *port)
	h.TLSName = *tlsName

	client, err := as.NewClientWithPolicyAndHost(policy, h)
	if err != nil {
		log.Fatalf("connect aerospike (tls): %v", err)
	}
	defer client.Close()

	// Pre-warm the pool. A burst of N parallel ops on a cold pool
	// has every goroutine racing to open its own TLS connection
	// through Keploy's proxy at replay time. The proxy is fast but
	// the kernel-level TLS handshakes serialise enough to push some
	// opens past Policy.Timeout, which surfaces as MAX_RETRIES_
	// EXCEEDED at the application and "mock not found" at the
	// proxy (closing bytes on a connection the client already gave
	// up on). Issuing a short sequential burst here populates the
	// pool with hot, reusable connections before the HTTP server
	// accepts the first /parallel request, so the burst hits a
	// warm pool and never tries to open more than a handful of
	// fresh sockets in parallel.
	// warmupPool floor must exceed the largest /parallel?n=N burst
	// we expect to handle. Anything below the burst size leaves a
	// window where the tend goroutine + an in-flight previous burst
	// can briefly hold the pool's idle count under N — and the one
	// goroutine that loses that race surfaces as MAX_RETRIES on
	// replay even though every other peer in its burst succeeds.
	// 64 lands comfortably above the documented /parallel cap of
	// 128's mid-range working set (n<=32 + tend + previous burst's
	// in-flight returns) without blowing through Keploy's TLS-MITM
	// throughput at warmup time.
	if err := warmupPool(client, 32); err != nil {
		log.Printf("warmupPool: %v (non-fatal)", err)
	}

	// Multi-client pool: a small bank of *as.Client instances each
	// with its own tend goroutine + connection pool. The /multiclient
	// handler round-robins HTTP requests across them — modelling a
	// service that, for example, runs one client per namespace or
	// per credential profile.
	//
	// We deliberately do NOT call warmupPool here. Five clients each
	// warming up in parallel produces hundreds of concurrent TLS
	// dials at startup, which the stunnel/socat fork model in the
	// compose stack can't accept fast enough — record then dies with
	// EOFs on the upstream dial path before any test runs. The
	// /multiclient handler instead uses parallelDo (10ms backoff,
	// 5 attempts) to ride out a cold pool on first contact, and the
	// per-op SleepBetweenRetries inside the policy gives the pool
	// time to recycle between attempts.
	multiClients := make([]*as.Client, 4)
	for i := range multiClients {
		mc, err := as.NewClientWithPolicyAndHost(policy, h)
		if err != nil {
			log.Fatalf("multi-client %d: %v", i, err)
		}
		multiClients[i] = mc
	}
	defer func() {
		for _, mc := range multiClients {
			mc.Close()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(client))
	mux.HandleFunc("/put", putHandler(client))
	mux.HandleFunc("/get/", getHandler(client))
	mux.HandleFunc("/batch/put", batchPutHandler(client))
	mux.HandleFunc("/batch/get", batchGetHandler(client))
	mux.HandleFunc("/scan", scanHandler(client))
	mux.HandleFunc("/query", queryHandler(client))
	mux.HandleFunc("/udf", udfHandler(client))
	mux.HandleFunc("/cdt/list/append", cdtListAppendHandler(client))
	mux.HandleFunc("/cdt/map/put", cdtMapPutHandler(client))
	mux.HandleFunc("/touch/", touchHandler(client))
	mux.HandleFunc("/key/", deleteHandler(client))
	mux.HandleFunc("/parallel", parallelHandler(client))
	mux.HandleFunc("/multiclient", multiClientHandler(multiClients))
	mux.HandleFunc("/freshclient", freshClientHandler(policy, h))

	log.Printf("aerospike-tls sample listening on %s (server %s:%d tls-name=%s)",
		*listen, *host, *port, *tlsName)
	srv := &http.Server{Addr: *listen, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func buildTLSConfig(caFile, certFile, keyFile, tlsName string, insecure bool) (*tls.Config, error) {
	cfg := &tls.Config{
		ServerName:         tlsName,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecure,
	}
	if caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read CA %q: %w", caFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("CA %q: no certs parsed", caFile)
		}
		cfg.RootCAs = pool
	}
	if certFile != "" && keyFile != "" {
		pair, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load client keypair: %w", err)
		}
		cfg.Certificates = []tls.Certificate{pair}
	}
	return cfg, nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return def
	}
	return n
}

func envBool(k string, def bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "t", "true", "y", "yes":
		return true
	case "0", "f", "false", "n", "no":
		return false
	}
	return def
}

func healthHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		nodes := c.GetNodes()
		writeJSON(w, http.StatusOK, map[string]any{
			"nodes":      len(nodes),
			"namespaces": "test",
		})
	}
}

type putReq struct {
	Key  string         `json:"key"`
	Bins map[string]any `json:"bins"`
}

func putHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body putReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		k, _ := as.NewKey("test", "demo", body.Key)
		bins := as.BinMap{}
		for n, v := range body.Bins {
			bins[n] = v
		}
		if err := c.Put(nil, k, bins); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func getHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/get/")
		k, _ := as.NewKey("test", "demo", key)
		rec, err := c.Get(nil, k)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, rec.Bins)
	}
}

func batchPutHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body []putReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, p := range body {
			k, _ := as.NewKey("test", "demo", p.Key)
			bins := as.BinMap{}
			for n, v := range p.Bins {
				bins[n] = v
			}
			if err := c.Put(nil, k, bins); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]int{"written": len(body)})
	}
}

func batchGetHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keys := r.URL.Query()["k"]
		batch := make([]*as.Key, 0, len(keys))
		for _, k := range keys {
			ak, _ := as.NewKey("test", "demo", k)
			batch = append(batch, ak)
		}
		recs, err := c.BatchGet(nil, batch)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			if r == nil {
				out = append(out, nil)
			} else {
				out = append(out, r.Bins)
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func scanHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		rs, err := c.ScanAll(nil, "test", "demo")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		count := 0
		for range rs.Results() {
			count++
		}
		writeJSON(w, http.StatusOK, map[string]int{"scanned": count})
	}
}

func queryHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		stmt := as.NewStatement("test", "demo")
		_ = stmt.SetFilter(as.NewRangeFilter("age", 0, 99))
		rs, err := c.Query(nil, stmt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		count := 0
		for range rs.Results() {
			count++
		}
		writeJSON(w, http.StatusOK, map[string]int{"matched": count})
	}
}

func udfHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body putReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		k, _ := as.NewKey("test", "demo", body.Key)
		v, err := c.Execute(nil, k, "transform", "apply", as.NewValue("bin"), as.NewValue(1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"result": v})
	}
}

func cdtListAppendHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body putReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		k, _ := as.NewKey("test", "demo", body.Key)
		_, err := c.Operate(nil, k,
			as.ListAppendOp("items", body.Bins["value"]),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "appended"})
	}
}

func cdtMapPutHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body putReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		k, _ := as.NewKey("test", "demo", body.Key)
		mapKey := body.Bins["mapKey"]
		mapVal := body.Bins["mapVal"]
		_, err := c.Operate(nil, k,
			as.MapPutOp(as.DefaultMapPolicy(), "mapBin", mapKey, mapVal),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "put"})
	}
}

func touchHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/touch/")
		k, _ := as.NewKey("test", "demo", key)
		if err := c.Touch(nil, k); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "touched"})
	}
}

func deleteHandler(c *as.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/key/")
		k, _ := as.NewKey("test", "demo", key)
		ok, err := c.Delete(nil, k)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": ok})
	}
}

// parallelHandler fans out N goroutines that each PUT and GET a unique
// key concurrently. The point is to stress Keploy's proxy with many
// simultaneous Aerospike connections from a single application, since
// the shared aerospike-Go client opens fresh sockets per concurrent op.
//
//	POST /parallel?n=8&prefix=p
type parallelResult struct {
	Key   string         `json:"key"`
	Bins  map[string]any `json:"bins,omitempty"`
	Error string         `json:"error,omitempty"`
}

// parallelWritePolicy and parallelReadPolicy are the per-op policies
// the /parallel handler uses. They differ from the defaults in two
// ways that matter under Keploy replay:
//
//  1. SocketTimeout and TotalTimeout are bumped well past what a
//     warm pool needs, so a single op can wait out a transient
//     pool-acquire stall instead of erroring with TIMEOUT.
//  2. MaxRetries is raised so a transient NO_AVAILABLE_CONNECTIONS_
//     TO_NODE on op-start retries against the now-warmer pool
//     instead of bubbling MAX_RETRIES_EXCEEDED to the user.
//
// Together with warmupPool, this gives every parallel worker a
// deterministic acquire-then-execute path even when N is several
// times the pool's resident size.
func parallelWritePolicy() *as.WritePolicy {
	p := as.NewWritePolicy(0, 0)
	p.SocketTimeout = 10 * time.Second
	p.TotalTimeout = 30 * time.Second
	p.MaxRetries = 10
	p.SleepBetweenRetries = 5 * time.Millisecond
	return p
}

func parallelReadPolicy() *as.BasePolicy {
	p := as.NewPolicy()
	p.SocketTimeout = 10 * time.Second
	p.TotalTimeout = 30 * time.Second
	p.MaxRetries = 10
	p.SleepBetweenRetries = 5 * time.Millisecond
	return p
}

// parallelDo runs op with up to `attempts` application-level retries
// on top of whatever in-client retry policy.MaxRetries already does.
// The client's retry path can still bubble MAX_RETRIES_EXCEEDED out
// when every retry hits an empty pool on its own goroutine slice;
// wrapping the whole PUT/GET in this outer loop gives the pool a
// few extra milliseconds — across cooperative goroutines — to
// recycle connections returned by peers in the same burst. Each
// outer attempt is its own clean acquire/execute cycle.
func parallelDo(attempts int, backoff time.Duration, op func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = op(); err == nil {
			return nil
		}
		time.Sleep(backoff)
	}
	return err
}

func parallelHandler(c *as.Client) http.HandlerFunc {
	wp := parallelWritePolicy()
	rp := parallelReadPolicy()
	return func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		if n <= 0 {
			n = 8
		}
		if n > 128 {
			n = 128
		}
		prefix := r.URL.Query().Get("prefix")
		if prefix == "" {
			prefix = "p"
		}

		out := make([]parallelResult, n)
		var wg sync.WaitGroup
		wg.Add(n)
		start := time.Now()
		for i := 0; i < n; i++ {
			i := i
			go func() {
				defer wg.Done()
				key := fmt.Sprintf("%s-%d", prefix, i)
				k, _ := as.NewKey("test", "demo", key)
				bins := as.BinMap{"idx": i, "tag": prefix}
				if err := parallelDo(5, 10*time.Millisecond, func() error {
					return c.Put(wp, k, bins)
				}); err != nil {
					out[i] = parallelResult{Key: key, Error: "put: " + err.Error()}
					return
				}
				var rec *as.Record
				if err := parallelDo(5, 10*time.Millisecond, func() error {
					var err error
					rec, err = c.Get(rp, k)
					return err
				}); err != nil {
					out[i] = parallelResult{Key: key, Error: "get: " + err.Error()}
					return
				}
				out[i] = parallelResult{Key: key, Bins: rec.Bins}
			}()
		}
		wg.Wait()
		writeJSON(w, http.StatusOK, map[string]any{
			"workers":  n,
			"prefix":   prefix,
			"duration": time.Since(start).String(),
			"results":  out,
		})
	}
}

// multiClientHandler round-robins each goroutine across a fixed
// bank of pre-built *as.Client instances. The point is a different
// concurrency shape than /parallel: with one shared client, all N
// goroutines compete for one pool's connect-or-wait latch; here,
// each client has its own pool + tend goroutine, so traffic fans
// out across len(clients) independent state machines simultaneously.
//
//	POST /multiclient?n=8&prefix=mc
//
// `n` is the goroutine count (cap 128). Goroutine i uses
// clients[i % len(clients)].
func multiClientHandler(clients []*as.Client) http.HandlerFunc {
	wp := parallelWritePolicy()
	rp := parallelReadPolicy()
	return func(w http.ResponseWriter, r *http.Request) {
		if len(clients) == 0 {
			http.Error(w, "no clients configured", http.StatusInternalServerError)
			return
		}
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		if n <= 0 {
			n = 8
		}
		if n > 128 {
			n = 128
		}
		prefix := r.URL.Query().Get("prefix")
		if prefix == "" {
			prefix = "mc"
		}

		out := make([]parallelResult, n)
		var wg sync.WaitGroup
		wg.Add(n)
		start := time.Now()
		for i := 0; i < n; i++ {
			i := i
			c := clients[i%len(clients)]
			go func() {
				defer wg.Done()
				key := fmt.Sprintf("%s-%d", prefix, i)
				k, _ := as.NewKey("test", "demo", key)
				bins := as.BinMap{"idx": i, "tag": prefix}
				if err := parallelDo(5, 10*time.Millisecond, func() error {
					return c.Put(wp, k, bins)
				}); err != nil {
					out[i] = parallelResult{Key: key, Error: "put: " + err.Error()}
					return
				}
				var rec *as.Record
				if err := parallelDo(5, 10*time.Millisecond, func() error {
					var err error
					rec, err = c.Get(rp, k)
					return err
				}); err != nil {
					out[i] = parallelResult{Key: key, Error: "get: " + err.Error()}
					return
				}
				out[i] = parallelResult{Key: key, Bins: rec.Bins}
			}()
		}
		wg.Wait()
		writeJSON(w, http.StatusOK, map[string]any{
			"workers":  n,
			"prefix":   prefix,
			"clients":  len(clients),
			"duration": time.Since(start).String(),
			"results":  out,
		})
	}
}

// freshClientHandler is the most aggressive variant: each goroutine
// constructs its OWN *as.Client inside the request, runs PUT+GET on
// it, then closes it. That means every burst of size N triggers N
// independent cluster bootstraps through Keploy's proxy in parallel
// — N copies of build/node/peers-clear-std/partition-generation
// before any data op flies.
//
//	POST /freshclient?n=4&prefix=fc
//
// We cap `n` lower than /multiclient because the per-request
// bootstrap is heavy and we want the recording to stay finite.
// `freshClientConcurrency` is the in-flight client-construction
// budget — beyond this, additional goroutines wait their turn.
// Without the cap the proxy's session-tier matcher sees too many
// simultaneous discovery handshakes at replay time and TLS handshake
// throughput becomes the bottleneck.
const freshClientConcurrency = 4

func freshClientHandler(policy *as.ClientPolicy, h *as.Host) http.HandlerFunc {
	wp := parallelWritePolicy()
	rp := parallelReadPolicy()
	sem := make(chan struct{}, freshClientConcurrency)
	return func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		if n <= 0 {
			n = 4
		}
		if n > 16 {
			n = 16
		}
		prefix := r.URL.Query().Get("prefix")
		if prefix == "" {
			prefix = "fc"
		}

		out := make([]parallelResult, n)
		var wg sync.WaitGroup
		wg.Add(n)
		start := time.Now()
		for i := 0; i < n; i++ {
			i := i
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				key := fmt.Sprintf("%s-%d", prefix, i)
				// Each goroutine builds its own client. NewClient...
				// performs cluster discovery synchronously, so the
				// returned client is already past bootstrap before we
				// issue any data op.
				c, err := as.NewClientWithPolicyAndHost(policy, h)
				if err != nil {
					out[i] = parallelResult{Key: key, Error: "newclient: " + err.Error()}
					return
				}
				defer c.Close()
				k, _ := as.NewKey("test", "demo", key)
				bins := as.BinMap{"idx": i, "tag": prefix}
				if err := parallelDo(5, 10*time.Millisecond, func() error {
					return c.Put(wp, k, bins)
				}); err != nil {
					out[i] = parallelResult{Key: key, Error: "put: " + err.Error()}
					return
				}
				var rec *as.Record
				if err := parallelDo(5, 10*time.Millisecond, func() error {
					var err error
					rec, err = c.Get(rp, k)
					return err
				}); err != nil {
					out[i] = parallelResult{Key: key, Error: "get: " + err.Error()}
					return
				}
				out[i] = parallelResult{Key: key, Bins: rec.Bins}
			}()
		}
		wg.Wait()
		writeJSON(w, http.StatusOK, map[string]any{
			"workers":     n,
			"prefix":      prefix,
			"concurrency": freshClientConcurrency,
			"duration":    time.Since(start).String(),
			"results":     out,
		})
	}
}

// warmupPool primes the aerospike-Go connection pool so a later
// /parallel burst hits idle connections instead of racing to open
// fresh TLS sockets through Keploy's proxy. It runs in two phases:
//
//  1. A short SEQUENTIAL prelude. The aerospike-Go client returns a
//     used connection to the pool, so a sequential loop only ever
//     keeps one connection resident — but it's enough to walk the
//     proxy through its first few TLS handshakes from a cold start.
//     Without this, the very first parallel batch below would itself
//     hit the cold-proxy thundering herd we're trying to avoid.
//
//  2. A PARALLEL fill of size `n`. Each goroutine grabs a distinct
//     pooled slot, and because the proxy is now warm, the N TLS
//     handshakes mostly stagger by microseconds and all complete.
//     After the wait, the pool holds up to `n` idle connections that
//     the /parallel handler can reuse without ever opening a new
//     socket — which is the property that makes replay deterministic
//     under bursts of size <= n.
func warmupPool(c *as.Client, n int) error {
	if n <= 0 {
		return nil
	}
	pol := parallelReadPolicy()
	// Phase 1 — sequential prelude. Eight ops is enough to put the
	// proxy past its cold-start serialisation; more would just be
	// wall-clock time wasted.
	const prelude = 8
	for i := 0; i < prelude; i++ {
		k, _ := as.NewKey("test", "demo", fmt.Sprintf("warmup-seq-%d", i))
		if _, err := c.Exists(pol, k); err != nil {
			return err
		}
	}
	// Phase 2 — parallel fill. `n` goroutines on distinct keys so the
	// pool ends up with `n` resident connections (one per goroutine).
	errs := make(chan error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			k, _ := as.NewKey("test", "demo", fmt.Sprintf("warmup-par-%d", i))
			if _, err := c.Exists(pol, k); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
