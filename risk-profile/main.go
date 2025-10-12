package main

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./risk.proto

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/keploy/samples-go/risk-profile/risk"

	"google.golang.org/grpc"
)

// --- V1 Data Structures and Data ---
type UserV1 struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var originalUsers = []UserV1{
	{ID: 1, Name: "Alice", Email: "alice@example.com"},
}

// --- API Handlers (V1) ---

func getUsersLowRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := originalUsers[0]
	response := map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func getUsersMediumRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := originalUsers[0]
	response := map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func getUsersMediumRiskWithAddition(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := originalUsers[0]
	response := map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func getUsersHighRiskType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := originalUsers[0]
	response := map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func getUsersHighRiskRemoval(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := originalUsers[0]
	response := map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func statusChangeHighRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":    "OK",
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func contentTypeChangeHighRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message":   "This is JSON.",
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func headerChangeMediumRisk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Custom-Header", "initial-value-123")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":    "header test",
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

func statusBodyChange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message":   "Status and body not changed",
		"timestamp": time.Now().UnixNano(),
	}
	json.NewEncoder(w).Encode(response)
}

func headerBodyChange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Transaction-ID", "txn-1")
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message":   "Header and body not changed",
		"timestamp": time.Now().UnixNano(),
	}
	json.NewEncoder(w).Encode(response)
}

func statusBodyHeaderChange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Transaction-ID", "txn-1")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message":   "Status, body, and header not changed",
		"timestamp": time.Now().UnixNano(),
	}
	json.NewEncoder(w).Encode(response)
}

// --- gRPC Server Implementation (V1) ---
type riskServer struct {
	pb.UnimplementedRiskServiceServer
}

func (s *riskServer) GetUserLowRisk(ctx context.Context, in *pb.Empty) (*pb.UserLowRisk, error) {
	return &pb.UserLowRisk{Id: 1, Name: "Alice", Email: "alice@example.com", Timestamp: time.Now().Unix()}, nil
}

func (s *riskServer) GetUserMediumRisk(ctx context.Context, in *pb.Empty) (*pb.UserMediumRisk, error) {
	return &pb.UserMediumRisk{Id: 1, Name: "Alice", Email: "alice@example.com", Timestamp: time.Now().Unix()}, nil
}

func (s *riskServer) GetUserMediumRiskWithAddition(ctx context.Context, in *pb.Empty) (*pb.UserMediumRiskWithAddition, error) {
	return &pb.UserMediumRiskWithAddition{Id: 1, Name: "Alice", Email: "alice@example.com", Timestamp: time.Now().Unix()}, nil
}

func (s *riskServer) GetUserHighRiskType(ctx context.Context, in *pb.Empty) (*pb.UserHighRiskType, error) {
	return &pb.UserHighRiskType{Id: 1, Name: "Alice", Email: "alice@example.com", Timestamp: time.Now().Unix()}, nil
}

func (s *riskServer) GetUserHighRiskRemoval(ctx context.Context, in *pb.Empty) (*pb.UserHighRiskRemoval, error) {
	return &pb.UserHighRiskRemoval{Id: 1, Name: "Alice", Email: "alice@example.com", Timestamp: time.Now().Unix(), RemovalRequested: true}, nil
}

func main() {
	log.Println("Application starting...")

	// --- Start gRPC Server ---
	grpcLis, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("failed to listen for gRPC: %v", err)
	}
	grpcServer := grpc.NewServer()

	pb.RegisterRiskServiceServer(grpcServer, &riskServer{})
	log.Printf("gRPC server listening at %v", grpcLis.Addr())
	go func() {
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// --- Start HTTP Server ---
	mux := http.NewServeMux()
	mux.HandleFunc("/users-low-risk", getUsersLowRisk)
	mux.HandleFunc("/users-medium-risk", getUsersMediumRisk)
	mux.HandleFunc("/users-medium-risk-with-addition", getUsersMediumRiskWithAddition)
	mux.HandleFunc("/users-high-risk-type", getUsersHighRiskType)
	mux.HandleFunc("/users-high-risk-removal", getUsersHighRiskRemoval)
	mux.HandleFunc("/status-change-high-risk", statusChangeHighRisk)
	mux.HandleFunc("/content-type-change-high-risk", contentTypeChangeHighRisk)
	mux.HandleFunc("/header-change-medium-risk", headerChangeMediumRisk)
	mux.HandleFunc("/status-body-change", statusBodyChange)
	mux.HandleFunc("/header-body-change", headerBodyChange)
	mux.HandleFunc("/status-body-header-change", statusBodyHeaderChange)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	log.Printf("HTTP server starting on port 8080...")
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("could not start HTTP server: %s\n", err)
		}
	}()

	// --- Graceful Shutdown Logic ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown signal received, initiating graceful shutdown...")

	// --- Shutdown Servers ---
	log.Println("Shutting down gRPC server...")
	grpcServer.GracefulStop()
	log.Println("gRPC server stopped.")

	log.Println("Shutting down HTTP server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server forced to shutdown: %v", err)
	}
	log.Println("HTTP server stopped.")

	log.Println("Application shut down gracefully.")
}
