package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	// Simple client to request the /big endpoint and print the response size.
	url := "http://localhost:8080/big"
	const totalRequests = 100

	for i := 1; i <= totalRequests; i++ {
		// retry a few times in case server isn't up yet
		var resp *http.Response
		var err error
		for j := 0; j < 5; j++ {
			resp, err = http.Get(url)
			if err == nil {
				break
			}
			log.Printf("request %d failed: %v, retrying...", i, err)
			time.Sleep(10 * time.Millisecond)
		}
		if err != nil {
			log.Fatalf("failed to GET %s on request %d: %v", url, i, err)
		}

		// Read and discard the body, then close immediately (don't defer in a loop).
		n, err := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatalf("error reading response for request %d: %v", i, err)
		}

		fmt.Printf("request %d: received %d bytes (status: %s)\n", i, n, resp.Status)
	}
}
