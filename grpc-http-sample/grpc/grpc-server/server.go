package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/Sarthak160/grpc-http-sample/grpc/grpc-server/example"
	"google.golang.org/grpc"
)

type server struct {
	example.UnimplementedExampleServiceServer
}

func (s *server) SayHello(ctx context.Context, req *example.HelloRequest) (*example.HelloResponse, error) {
	message := fmt.Sprintf("Hello, %s!", req.Name)
	return &example.HelloResponse{Message: message}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	example.RegisterExampleServiceServer(s, &server{})
	log.Println("gRPC server running on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
