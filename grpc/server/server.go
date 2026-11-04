package server

import (
	"context"
	"io"
	"log"
	"net"

	pb "github.com/keploy/samples-go/grpc/generated/proto"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedStudentServiceServer
}

func (s *server) GetStudent(ctx context.Context, req *pb.StudentRequest) (*pb.StudentResponse, error) {
	log.Printf("Received: %v", req.GetId())
	return &pb.StudentResponse{Name: "John Doe", Age: 21}, nil
}

func (s *server) GetStudentStream(req *pb.StudentRequest, stream pb.StudentService_GetStudentStreamServer) error {
	// For simplicity, just sending a static response.
	err := stream.Send(&pb.StudentResponse{Name: "John Doe", Age: 21})
	if err != nil {
		return err
	}
	return nil
}

func (s *server) SendStudentStream(stream pb.StudentService_SendStudentStreamServer) error {
	var student *pb.StudentRequest
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.StudentResponse{Name: "Received all requests", Age: 0})
		}
		if err != nil {
			return err
		}
		student = req
		log.Printf("Received: %v", student.GetId())
	}
}

func (s *server) SendAndGetStudentStream(stream pb.StudentService_SendAndGetStudentStreamServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		log.Printf("Received: %v", req.GetId())
		if err := stream.Send(&pb.StudentResponse{Name: "John Doe", Age: 21}); err != nil {
			return err
		}
	}
}

func StartServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterStudentServiceServer(s, &server{})

	log.Printf("Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
