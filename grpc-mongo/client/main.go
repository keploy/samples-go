package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	pb "grpc-mongo/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := os.Getenv("GRPC_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}

	// dial with timeout
	dialCtx, cancelDial := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDial()
	conn, err := grpc.DialContext(
		dialCtx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	c := pb.NewTokenServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Seed 10 tokens (order matters, first popped = first returned)
	seed := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta", "iota", "kappa"}
	ack, err := c.SeedTokens(ctx, &pb.SeedRequest{Tokens: seed})
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	fmt.Println("Seed:", ack.Message)

	// Make 10 identical requests -> 10 different replies
	for i := 1; i <= 10; i++ {
		r, err := c.NextToken(ctx, &pb.NextTokenRequest{})
		if err != nil {
			log.Fatalf("NextToken #%d: %v", i, err)
		}
		fmt.Printf("NextToken #%d: %s\n", i, r.Token)
	}
}
