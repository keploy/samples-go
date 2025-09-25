// Package main implements a gRPC server for managing users.
// It supports various gRPC methods for creating, updating, deleting, and retrieving users,
// including both synchronous and streaming RPCs for handling multiple users in one request.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	pb "github.com/keploy/samples-go/go-grpc/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
)

var (
	userStore = make(map[int]User) // In-memory store
	mu        sync.Mutex           // Mutex to ensure thread-safety
	idCounter = 0                  // Global counter for unique IDs
)

type User struct {
	ID    int    `json:"id,omitempty"`
	Name  string `json:"name" bson:"name"`
	Email string `json:"email" bson:"email"`
	Age   int32  `json:"age" bson:"age"`
}

type server struct {
	pb.UnimplementedUserServiceServer
}

func incrementID() {
	idCounter++
}

// CreateUser RPC
func (s *server) CreateUser(_ context.Context, req *pb.UserRequest) (*pb.UserResponse, error) {
	mu.Lock()
	defer mu.Unlock()

	log.Println("gRPC server received a request to create a user")

	incrementID()

	user := &User{
		ID:    idCounter,
		Name:  req.GetName(),
		Email: req.GetEmail(),
		Age:   req.GetAge(),
	}

	userStore[user.ID] = *user

	return &pb.UserResponse{User: &pb.User{
		Id:    int32(user.ID),
		Name:  user.Name,
		Email: user.Email,
		Age:   int32(user.Age),
	}}, nil
}

// GetUsers RPC
func (s *server) GetUsers(_ context.Context, _ *pb.Empty) (*pb.UsersResponse, error) {
	mu.Lock()
	defer mu.Unlock()

	log.Println("gRPC server received a request to get all users")

	var users []*pb.User
	for _, user := range userStore {
		users = append(users, &pb.User{
			Id:    int32(user.ID),
			Name:  user.Name,
			Email: user.Email,
			Age:   int32(user.Age),
		})
	}

	return &pb.UsersResponse{Users: users}, nil
}

// UpdateUser RPC
func (s *server) UpdateUser(_ context.Context, req *pb.UserRequest) (*pb.UserResponse, error) {
	mu.Lock()
	defer mu.Unlock()

	log.Println("gRPC server received a request to update the user")

	if user, exists := userStore[int(req.GetId())]; exists {
		user.Name = req.GetName()
		user.Email = req.GetEmail()
		user.Age = req.GetAge()
		userStore[int(req.GetId())] = user
		return &pb.UserResponse{User: &pb.User{
			Id:    int32(user.ID),
			Name:  user.Name,
			Email: user.Email,
			Age:   int32(user.Age),
		}}, nil
	}
	return nil, fmt.Errorf("user not found")
}

// DeleteUser RPC
func (s *server) DeleteUser(_ context.Context, req *pb.UserID) (*pb.Empty, error) {
	mu.Lock()
	defer mu.Unlock()

	log.Println("gRPC server received a request to delete the user")

	if _, exists := userStore[int(req.GetId())]; exists {
		delete(userStore, int(req.GetId()))
		return &pb.Empty{}, nil
	}
	return nil, fmt.Errorf("user not found")
}

// CreateUsersStream RPC (Client Streaming)
func (s *server) CreateUsersStream(stream pb.UserService_CreateUsersStreamServer) error {
	mu.Lock()
	defer mu.Unlock()

	log.Println("gRPC server received a request to create the users in stream")

	gotUsers := make([]*pb.User, 0)

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.UsersResponse{Users: gotUsers})
		}
		if err != nil {
			return err
		}

		incrementID()
		user := &User{
			ID:    idCounter,
			Name:  req.GetName(),
			Email: req.GetEmail(),
			Age:   req.GetAge(),
		}

		userStore[user.ID] = *user

		gotUsers = append(gotUsers, &pb.User{
			Id:    int32(user.ID),
			Name:  user.Name,
			Email: user.Email,
			Age:   int32(user.Age),
		})
	}
}

// DeleteUsersStream RPC (Client Streaming)
func (s *server) DeleteUsersStream(stream pb.UserService_DeleteUsersStreamServer) error {
	mu.Lock()
	defer mu.Unlock()

	log.Println("gRPC server received a request to delete the users in stream")

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.Empty{})
		}
		if err != nil {
			return err
		}

		delete(userStore, int(req.GetId()))
	}
}

func (s *server) GetLargeUsersResponse(_ context.Context, _ *pb.Empty) (*pb.UsersResponse, error) {
	log.Println("gRPC server received a request to get a large user payload")

	const targetSize = 3 * 1024 * 1024 // 3 MB in bytes

	// Create a sample user to measure its serialized size.
	// Using realistic but long strings to get a decent base size.
	sampleUser := &pb.User{
		Id:    1,
		Name:  "Johnathan Doe The Third, Esquire of the Grand Valley Region",
		Email: "johnathan.doe.the.third.esquire.of.the.grand.valley.region@example.com",
		Age:   42,
	}

	// Marshal the sample user to find its approximate size in bytes.
	sampleUserBytes, err := proto.Marshal(sampleUser)
	if err != nil {
		return nil, fmt.Errorf("failed to measure user size: %w", err)
	}
	sizePerUser := len(sampleUserBytes)

	// Calculate how many users we need to reach the target size.
	numUsers := targetSize / sizePerUser
	if targetSize%sizePerUser != 0 {
		numUsers++ // Add one more to ensure we are at or over the target size
	}

	// Generate the list of users.
	users := make([]*pb.User, 0, numUsers)
	for i := 0; i < numUsers; i++ {
		users = append(users, &pb.User{
			Id:    int32(i + 1), // Unique ID for each user
			Name:  sampleUser.Name,
			Email: sampleUser.Email,
			Age:   sampleUser.Age,
		})
	}

	log.Printf("Generated %d users with an approximate total payload size of %.2f MB", numUsers, float64(numUsers*sizePerUser)/1024/1024)

	return &pb.UsersResponse{Users: users}, nil
}

// GetUsersStream RPC (Server Streaming)
func (s *server) GetUsersStream(_ *pb.Empty, stream pb.UserService_GetUsersStreamServer) error {
	mu.Lock()
	defer mu.Unlock()

	log.Println("gRPC server received a request to get the users in stream")

	for _, user := range userStore {
		if err := stream.Send(&pb.UserResponse{
			User: &pb.User{
				Id:    int32(user.ID),
				Name:  user.Name,
				Email: user.Email,
				Age:   int32(user.Age),
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

// UpdateUsersStream RPC (Bi-Directional Streaming)
func (s *server) UpdateUsersStream(stream pb.UserService_UpdateUsersStreamServer) error {
	mu.Lock()
	defer mu.Unlock()

	log.Println("gRPC server received a request to update the users in stream")

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}

		req, err := stream.Recv()
		if err == io.EOF {
			// close stream from server side
			return nil
		}
		if err != nil {
			return err
		}

		// Update user in the in-memory store
		if existingUser, exists := userStore[int(req.GetId())]; exists {
			existingUser.Name = req.GetName()
			existingUser.Email = req.GetEmail()
			existingUser.Age = req.GetAge()
			userStore[int(req.GetId())] = existingUser

			// Send the updated user back to the client
			if err := stream.Send(&pb.UserResponse{
				User: &pb.User{
					Id:    int32(existingUser.ID),
					Name:  existingUser.Name,
					Email: existingUser.Email,
					Age:   int32(existingUser.Age),
				},
			}); err != nil {
				return err
			}
		} else {
			if err := stream.Send(&pb.UserResponse{
				User: nil,
			}); err != nil {
				return err
			}
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Printf("failed to listen: %v", err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, &server{})

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)

	healthServer.SetServingStatus("user.UserService", grpc_health_v1.HealthCheckResponse_SERVING)

	log.Println("gRPC server running on port 50051")
	if err := s.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
		os.Exit(1)
	}
}
