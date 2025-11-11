package main

import (
	"log"
	"net/http"
	"strconv"
)

func main() {
	// Pre-generate ~4 MiB payload
	const size = 4 * 1024 // 4 MiB
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = 'a'
	}

	http.HandleFunc("/big", func(w http.ResponseWriter, r *http.Request) {
		// Serve as binary data. Using Content-Length lets clients know size up front.
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		// Write payload in a single write (it's fine for this size).
		_, err := w.Write(payload)
		if err != nil {
			log.Printf("error writing response: %v", err)
		}
	})

	addr := ":8080"
	log.Printf("starting server on %s, /big -> ~%d bytes\n", addr, len(payload))
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
