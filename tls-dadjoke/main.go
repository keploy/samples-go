package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

// jokeHandler is the function that handles requests to the /joke endpoint.
func jokeHandler(w http.ResponseWriter, r *http.Request) {
	// The API URL for icanhazdadjoke.com
	jokeAPIURL := "https://icanhazdadjoke.com/"

	// Create a new HTTP client to make the request.
	client := &http.Client{}

	// Create a new GET request. We use NewRequest so we can set headers.
	req, err := http.NewRequest("GET", jokeAPIURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// The icanhazdadjoke.com API requires this header to send back JSON.
	// Without it, it will send back HTML.
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Go Joke App (https://github.com/example/repo)")


	// Execute the request.
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching joke: %v", err)
		http.Error(w, "Failed to fetch joke", http.StatusServiceUnavailable)
		return
	}
	// Ensure the response body is closed when the function returns.
	defer resp.Body.Close()

	// Check if the request to the joke API was successful.
	if resp.StatusCode != http.StatusOK {
		log.Printf("Joke API returned non-200 status: %d", resp.StatusCode)
		http.Error(w, "Joke service is down", http.StatusServiceUnavailable)
		return
	}

	// Read the body of the response from the joke API.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set the Content-Type header of our response to application/json.
	w.Header().Set("Content-Type", "application/json")

	// Write the joke (which is already in JSON format) to our response.
	w.Write(body)
}

func main() {
	// Register our jokeHandler function to handle all requests to the "/joke" path.
	http.HandleFunc("/joke", jokeHandler)

	port := "8080"
	fmt.Printf("Starting server on port %s\n", port)
	fmt.Printf("Visit http://localhost:%s/joke to get a dad joke!\n", port)

	// Start the web server on port 8080.
	// If ListenAndServe returns an error, we log it and exit.
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
