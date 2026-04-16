// Package main is a test app for Keploy DNS mock deduplication scenarios.
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)

// This app tests that Keploy properly deduplicates DNS mocks when the same
// domain returns different IPs on each lookup (round-robin / load-balancing).
//
// AWS services like SQS rotate IPs per DNS query. Before the fix, Keploy
// recorded a new DNS mock for every unique IP set, resulting in thousands of
// duplicate DNS mocks for a single domain.
//
// The /resolve-many endpoint triggers many DNS lookups for the same domain,
// which is the key scenario for verifying deduplication.

func main() {
	domain := "sqs.us-east-1.amazonaws.com"
	if len(os.Args) > 1 {
		domain = os.Args[1]
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprint(w, "ok"); err != nil {
			fmt.Fprintf(os.Stderr, "health write error: %v\n", err)
		}
	})

	// Single DNS lookup
	http.HandleFunc("/resolve", func(w http.ResponseWriter, r *http.Request) {
		d := r.URL.Query().Get("domain")
		if d == "" {
			d = domain
		}
		ips, err := net.LookupHost(d)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sort.Strings(ips) // deterministic order for replay comparison
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"domain": d,
			"ips":    ips,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "encode error: %v\n", err)
		}
	})

	// Many DNS lookups for the same domain — the key dedup scenario.
	// Without dedup, each unique IP set becomes a separate DNS mock.
	http.HandleFunc("/resolve-many", func(w http.ResponseWriter, r *http.Request) {
		d := r.URL.Query().Get("domain")
		if d == "" {
			d = domain
		}
		n := 20
		if ns := r.URL.Query().Get("n"); ns != "" {
			if parsed, err := strconv.Atoi(ns); err == nil && parsed > 0 {
				n = parsed
			}
		}

		seen := make(map[string]bool)
		type result struct {
			Iteration int      `json:"iteration"`
			IPs       []string `json:"ips,omitempty"`
			New       bool     `json:"new"`
			Error     string   `json:"error,omitempty"`
		}
		results := make([]result, 0, n)

		for i := 1; i <= n; i++ {
			ips, err := net.LookupHost(d)
			if err != nil {
				results = append(results, result{Iteration: i, Error: err.Error()})
			} else {
				sort.Strings(ips) // deterministic order for replay comparison
				key := fmt.Sprintf("%v", ips)
				isNew := !seen[key]
				seen[key] = true
				results = append(results, result{Iteration: i, IPs: ips, New: isNew})
			}
			time.Sleep(50 * time.Millisecond)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"domain":         d,
			"total_queries":  n,
			"unique_ip_sets": len(seen),
			"results":        results,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "encode error: %v\n", err)
		}
	})

	port := "8086"
	fmt.Printf("DNS dedup test server starting on :%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
