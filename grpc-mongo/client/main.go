// Package main implements the gRPC client for the grpc-mongo sample.
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
	conn, err := grpc.DialContext( //nolint:staticcheck
		dialCtx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), //nolint:staticcheck
	)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	c := pb.NewTokenServiceClient(conn)

	// Per-call deadlines: each RPC gets its own bounded context so the client's
	// deliberate inter-call pacing (and the slow proxyless-capture warmup on a
	// loaded CI runner) never eats into a later call's budget. A single shared
	// absolute 20s deadline conflated all 11 calls plus their 1s pacing sleeps
	// into one window, so under contention NextToken #9 sporadically tripped
	// "DeadlineExceeded". A genuinely hung RPC still times out at callTimeout.
	const callTimeout = 10 * time.Second

	// Seed 10 tokens (order matters, first popped = first returned)
	seed := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta", "iota", "kappa"}
	seedCtx, cancelSeed := context.WithTimeout(context.Background(), callTimeout)
	ack, err := c.SeedTokens(seedCtx, &pb.SeedRequest{Tokens: seed})
	cancelSeed()
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	fmt.Println("Seed:", ack.Message)
	time.Sleep(1 * time.Second)

	// Make 10 identical requests -> 10 different replies
	for i := 1; i <= 10; i++ {
		callCtx, cancelCall := context.WithTimeout(context.Background(), callTimeout)
		r, err := c.NextToken(callCtx, &pb.NextTokenRequest{})
		cancelCall()
		if err != nil {
			log.Fatalf("NextToken #%d: %v", i, err)
		}
		fmt.Printf("NextToken #%d: %s\n", i, r.Token)
		time.Sleep(1 * time.Second)
	}
}
