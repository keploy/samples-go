package main

import (
	"crypto/tls"
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
	proxyAddr := os.Getenv("HTTP_PROXY")
	if proxyAddr == "" {
		proxyAddr = os.Getenv("HTTPS_PROXY")
	}
	if proxyAddr == "" {
		http.Error(w, `{"error":"no HTTP_PROXY configured"}`, http.StatusInternalServerError)
		return
	}

	proxyURL, err := url.Parse(proxyAddr)
	if err != nil {
		http.Error(w, `{"error":"bad proxy url"}`, http.StatusInternalServerError)
		return
	}

	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "https://httpbin.org/get"
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 15 * time.Second,
	}

	resp, err := client.Get(targetURL)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var upstream map[string]interface{}
	if err := json.Unmarshal(body, &upstream); err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"proxied":      true,
		"upstream_url":  upstream["url"],
		"status_code":  resp.StatusCode,
	})
}
