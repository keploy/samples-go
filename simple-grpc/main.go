package main

import (
	"context"
	"log"
	"net"
	"time"

	"simple-grpc/pb" // Import the generated package

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedDemoServiceServer
}

// 1. Health Check
func (s *server) Health(ctx context.Context, in *pb.Empty) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Ok: true}, nil
}

// 2. Simple Hello
func (s *server) Hello(ctx context.Context, in *pb.Empty) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{Msg: "hello from grpc"}, nil
}

// 3. Echo
func (s *server) Echo(ctx context.Context, in *pb.EchoRequest) (*pb.EchoResponse, error) {
	msg := in.GetMsg()
	if msg == "" {
		msg = "empty"
	}
	return &pb.EchoResponse{Echo: msg}, nil
}

// 4. Add Numbers
func (s *server) Add(ctx context.Context, in *pb.AddRequest) (*pb.AddResponse, error) {
	sum := in.GetA() + in.GetB()
	return &pb.AddResponse{
		A:   in.GetA(),
		B:   in.GetB(),
		Sum: sum,
	}, nil
}

// 5. Get Current Time
func (s *server) GetTime(ctx context.Context, in *pb.Empty) (*pb.TimeResponse, error) {
	return &pb.TimeResponse{
		Now: time.Now().UTC().Format(time.RFC3339Nano),
	}, nil
}

// 6. Get Resource by ID
func (s *server) GetResource(ctx context.Context, in *pb.ResourceRequest) (*pb.ResourceResponse, error) {
	id := in.GetId()
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "missing id")
	}

	return &pb.ResourceResponse{
		Id:    id,
		Type:  "demo",
		Fixed: true,
	}, nil
}

func main() {
	// Listen on TCP port 50051 (Standard gRPC port)
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	// Register our implementation
	pb.RegisterDemoServiceServer(s, &server{})

	log.Printf("gRPC server listening on %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
