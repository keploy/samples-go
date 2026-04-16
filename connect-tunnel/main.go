// Package main implements a minimal HTTP app for testing Keploy's CONNECT tunnel recording and replay.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Minimal app for testing Keploy's CONNECT tunnel recording and replay.
//
// Endpoints:
//   GET /health       — returns {"status":"ok"}, no external deps
//   GET /via-proxy    — fetches an HTTPS URL through HTTP_PROXY/HTTPS_PROXY CONNECT tunnel

var proxyClient *http.Client

func init() {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment
	proxyClient = &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}
}

func main() {
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/via-proxy", handleViaProxy)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("connect-tunnel sample listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("server failed on port %s: %v (check if port is already in use or set APP_PORT)", port, err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("failed to write health response (likely client disconnect): %v", err)
	}
}

func handleViaProxy(w http.ResponseWriter, r *http.Request) {
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "https://httpbin.org/get"
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "invalid TARGET_URL; check the TARGET_URL environment variable")
		return
	}

	resp, err := proxyClient.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "upstream request failed; check proxy connectivity (HTTP_PROXY/HTTPS_PROXY), DNS resolution, and target reachability")
		return
	}
	defer resp.Body.Close() //nolint:errcheck

	const maxBody = 1 << 20 // 1 MiB
	lr := io.LimitReader(resp.Body, maxBody+1)
	body, err := io.ReadAll(lr)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to read upstream response body")
		return
	}
	if len(body) > maxBody {
		writeJSONError(w, http.StatusBadGateway, "upstream response exceeded 1 MiB limit")
		return
	}

	var upstream map[string]interface{}
	if err := json.Unmarshal(body, &upstream); err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(resp.StatusCode)
		if _, writeErr := w.Write(body); writeErr != nil {
			log.Printf("failed to write response body (likely client disconnect): %v", writeErr)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"upstream_url": upstream["url"],
		"status_code":  resp.StatusCode,
	}); err != nil {
		log.Printf("failed to encode response (likely client disconnect): %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		log.Printf("failed to write error response (likely client disconnect): %v", err)
	}
}
