package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Sarthak160/grpc-http-sample/grpc/grpc-server/example"
	"google.golang.org/grpc"
)

// Call the gRPC service from the HTTP handler
func callGRPCService(name string) (string, error) {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		return "", fmt.Errorf("could not connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := example.NewExampleServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := client.SayHello(ctx, &example.HelloRequest{Name: name})
	if err != nil {
		return "", fmt.Errorf("could not call SayHello: %v", err)
	}

	return response.Message, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "world"
	}

	message, err := callGRPCService(name)
	if err != nil {
		http.Error(w, "Failed to call gRPC service", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Response from gRPC: %s", message)
}

func main() {
	http.HandleFunc("/hello", handler)
	log.Println("HTTP server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
