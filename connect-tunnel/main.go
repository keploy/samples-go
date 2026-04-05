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
//   GET /via-proxy    — fetches an HTTPS URL through HTTP_PROXY CONNECT tunnel

var proxyClient *http.Client

func init() {
	proxyClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
		Timeout: 15 * time.Second,
	}

	if os.Getenv("HTTP_PROXY") == "" && os.Getenv("HTTPS_PROXY") == "" &&
		os.Getenv("http_proxy") == "" && os.Getenv("https_proxy") == "" {
		log.Println("no proxy environment variables set; /via-proxy will use direct connections instead of CONNECT tunnel")
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

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("failed to write health response: %v", err)
	}
}

func handleViaProxy(w http.ResponseWriter, r *http.Request) {
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "https://httpbin.org/get"
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create request")
		return
	}

	resp, err := proxyClient.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "upstream request failed")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to read upstream response")
		return
	}

	var upstream map[string]interface{}
	if err := json.Unmarshal(body, &upstream); err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(resp.StatusCode)
		if _, writeErr := w.Write(body); writeErr != nil {
			log.Printf("failed to write response body: %v", writeErr)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"proxied":     true,
		"upstream_url": upstream["url"],
		"status_code": resp.StatusCode,
	}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		log.Printf("failed to write error response: %v", err)
	}
}
