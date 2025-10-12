package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "github.com/keploy/samples-go/risk-profile/risk"
)

const (
	address = "localhost:8081"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewRiskServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	log.Println("Calling GetUserLowRisk...")
	_, err = c.GetUserLowRisk(ctx, &pb.Empty{})
	if err != nil {
		log.Printf("could not call GetUserLowRisk: %v", err)
	}

	log.Println("Calling GetUserMediumRisk...")
	_, err = c.GetUserMediumRisk(ctx, &pb.Empty{})
	if err != nil {
		log.Printf("could not call GetUserMediumRisk: %v", err)
	}

	log.Println("Calling GetUserHighRiskType...")
	_, err = c.GetUserHighRiskType(ctx, &pb.Empty{})
	if err != nil {
		log.Printf("could not call GetUserHighRiskType: %v", err)
	}

	log.Println("Calling GetUserHighRiskRemoval...")
	_, err = c.GetUserHighRiskRemoval(ctx, &pb.Empty{})
	if err != nil {
		log.Printf("could not call GetUserHighRiskRemoval: %v", err)
	}

	log.Println("Calling StatusChangeHighRisk...")
	_, err = c.StatusChangeHighRisk(ctx, &pb.Empty{})
	if err != nil {
		log.Printf("could not call StatusChangeHighRisk: %v", err)
	}

	log.Println("Finished gRPC client calls.")
}
