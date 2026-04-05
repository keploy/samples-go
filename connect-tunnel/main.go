package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Minimal app for testing Keploy's CONNECT tunnel recording and replay.
//
// Endpoints:
//   GET /health       — returns {"status":"ok"}, no external deps
//   GET /via-proxy    — fetches https://httpbin.org/get through HTTP_PROXY

var proxyClient *http.Client

func init() {
	proxyAddr := os.Getenv("HTTP_PROXY")
	if proxyAddr == "" {
		proxyAddr = os.Getenv("HTTPS_PROXY")
	}
	transport := &http.Transport{}
	if proxyAddr != "" {
		proxyURL, err := url.Parse(proxyAddr)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
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
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleViaProxy(w http.ResponseWriter, _ *http.Request) {
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "https://httpbin.org/get"
	}

	resp, err := proxyClient.Get(targetURL)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "upstream request failed", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to read upstream response", err)
		return
	}

	var upstream map[string]interface{}
	if err := json.Unmarshal(body, &upstream); err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"proxied":     true,
		"upstream_url": upstream["url"],
		"status_code": resp.StatusCode,
	})
}

func writeJSONError(w http.ResponseWriter, status int, msg string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   msg,
		"details": err.Error(),
	})
}
