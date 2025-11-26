package main

import (
	"context"
	"log"
	"net"
	"time"

	pb "github.com/keploy/samples-go/grpc-apps/multi-proto/protos/svc/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	port = ":50051"
)

type server struct {
	pb.UnimplementedDataServiceServer
}

func (s *server) GetSimpleData(ctx context.Context, req *pb.SimpleDataRequest) (*pb.SimpleDataResponse, error) {
	log.Printf("Received GetSimpleData request with ID: %s", req.Id)

	response := &pb.SimpleDataResponse{
		Message:   "Simple data retrieved successfully",
		Code:      200,
		Success:   true,
		Value:     123.45,
		Items:     []string{"item1", "item2", "item3"},
		Metadata:  map[string]string{"key1": "value1", "key2": "value2"},
		Timestamp: timestamppb.Now(),
	}

	return response, nil
}

func (s *server) GetComplexData(ctx context.Context, req *pb.ComplexDataRequest) (*pb.ComplexDataResponse, error) {
	log.Printf("Received GetComplexData request with query: %s, limit: %d", req.Query, req.Limit)

	// Create sample users
	users := []*pb.User{
		{
			Id:        "user-001",
			Name:      "John Doe",
			Email:     "john.doe@example.com",
			Age:       30,
			IsActive:  true,
			Balance:   1500.75,
			CreatedAt: timestamppb.New(time.Now().Add(-24 * time.Hour)),
			Address: &pb.Address{
				Street:  "123 Main St",
				City:    "New York",
				State:   "NY",
				ZipCode: "10001",
				Country: "USA",
			},
			Tags:     []string{"premium", "verified"},
			Metadata: map[string]string{"source": "web", "referrer": "google"},
		},
		{
			Id:        "user-002",
			Name:      "Jane Smith",
			Email:     "jane.smith@example.com",
			Age:       28,
			IsActive:  true,
			Balance:   2300.50,
			CreatedAt: timestamppb.New(time.Now().Add(-48 * time.Hour)),
			Address: &pb.Address{
				Street:  "456 Oak Ave",
				City:    "Los Angeles",
				State:   "CA",
				ZipCode: "90001",
				Country: "USA",
			},
			Tags:     []string{"new", "active"},
			Metadata: map[string]string{"source": "mobile", "referrer": "facebook"},
		},
	}

	// Create sample products
	products := []*pb.Product{
		{
			Id:          "prod-001",
			Name:        "Laptop",
			Description: "High-performance laptop",
			Price:       1299.99,
			Quantity:    50,
			InStock:     true,
			Categories:  []string{"electronics", "computers"},
			Attributes:  map[string]string{"brand": "TechCorp", "color": "silver"},
			UpdatedAt:   timestamppb.Now(),
			Details: &pb.ProductDetails{
				Manufacturer:   "TechCorp Inc.",
				Model:          "TC-2024",
				WarrantyMonths: 24,
				Weight:         2.5,
				Dimensions:     map[string]float64{"length": 35.5, "width": 24.5, "height": 2.0},
			},
		},
		{
			Id:          "prod-002",
			Name:        "Wireless Mouse",
			Description: "Ergonomic wireless mouse",
			Price:       29.99,
			Quantity:    200,
			InStock:     true,
			Categories:  []string{"electronics", "accessories"},
			Attributes:  map[string]string{"brand": "Peripherals Plus", "color": "black"},
			UpdatedAt:   timestamppb.Now(),
			Details: &pb.ProductDetails{
				Manufacturer:   "Peripherals Plus Ltd.",
				Model:          "WM-500",
				WarrantyMonths: 12,
				Weight:         0.1,
				Dimensions:     map[string]float64{"length": 10.0, "width": 6.0, "height": 4.0},
			},
		},
	}

	response := &pb.ComplexDataResponse{
		Status:      "success",
		TotalCount:  4,
		HasMore:     false,
		Users:       users,
		Products:    products,
		Statistics:  map[string]int32{"total_users": 2, "total_products": 2, "active_users": 2},
		ProcessedAt: timestamppb.Now(),
		Metadata: &pb.ResponseMetadata{
			RequestId:        "req-" + time.Now().Format("20060102150405"),
			ProcessingTimeMs: 125.5,
			ServerVersion:    "1.0.0",
			Tags:             map[string]string{"environment": "production", "region": "us-east-1"},
		},
	}

	return response, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterDataServiceServer(s, &server{})

	// Register reflection service on gRPC server
	reflection.Register(s)

	log.Printf("Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
