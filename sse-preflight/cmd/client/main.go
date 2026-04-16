// Package main is a CORS preflight client for the sse-preflight sample.
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	targetURL := flag.String("url", "http://localhost:8047/subscribe/student/events?doubtId=repro", "URL to send the CORS preflight to")
	hostHeader := flag.String("host", "", "Host header override (optional)")
	origin := flag.String("origin", "https://web.example.com", "Origin header")
	flag.Parse()

	req, err := http.NewRequest(http.MethodOptions, *targetURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create request: %v\n", err)
		os.Exit(1)
	}

	if *hostHeader != "" {
		req.Host = *hostHeader
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", *origin)
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type,x-client-type,x-device-id,x-source")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read response body: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("status=%s\n", resp.Status)
	fmt.Printf("headers=%v\n", resp.Header)
	fmt.Printf("body=%q\n", string(body))
}
