package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	pb "loadtestgrpcapi/api/proto"
	"loadtestgrpcapi/internal/config"
	"loadtestgrpcapi/internal/grpcapi"
	"loadtestgrpcapi/internal/store"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()
	st := store.New()
	srv := grpcapi.New(st)

	// gRPC server
	grpcLis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("grpc listen :%s: %v", cfg.GRPCPort, err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterLoadTestServiceServer(grpcServer, srv)
	reflection.Register(grpcServer)

	// HTTP server (health-check only)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})
	httpServer := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("gRPC server listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	go func() {
		log.Printf("HTTP server listening on :%s", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server stopped: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down…")
	grpcServer.GracefulStop()
	_ = httpServer.Shutdown(context.Background())
}
