package client

import (
	"context"
	"io"
	"log"
	"time"

	pb "github.com/keploy/samples-go/grpc/generated/proto"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50051"
)

func StartClient() {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewStudentServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	stream, err := client.GetStudentStream(ctx, &pb.StudentRequest{Id: 1})
	if err != nil {
		log.Fatalf("could not get student stream: %v", err)
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			log.Println("Stream closed")
			break
		}
		if err != nil {
			log.Fatalf("stream receive error: %v", err)
		}
		log.Printf("Streamed Student: Name: %s, Age: %d", msg.GetName(), msg.GetAge())
	}
}
