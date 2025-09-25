package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	pb "grpc-mongo/proto"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedTokenServiceServer
	col *mongo.Collection
}

func mongoConnect(ctx context.Context, uri string) (*mongo.Client, error) {
	clientOpts := options.Client().ApplyURI(uri).
		SetMinPoolSize(1).
		SetMaxPoolSize(1).
		SetRetryReads(false).
		SetRetryWrites(false).
		SetDirect(true).
		SetHeartbeatInterval(90 * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (s *server) SeedTokens(ctx context.Context, req *pb.SeedRequest) (*pb.Ack, error) {
	if len(req.Tokens) == 0 {
		return &pb.Ack{Ok: false, Message: "no tokens provided"}, nil
	}
	// wipe collection
	if _, err := s.col.DeleteMany(ctx, bson.D{}); err != nil {
		return nil, status.Errorf(codes.Internal, "deleteMany: %v", err)
	}
	// insert in order
	docs := make([]interface{}, 0, len(req.Tokens))
	for _, t := range req.Tokens {
		docs = append(docs, bson.D{{Key: "token", Value: t}, {Key: "ts", Value: time.Now()}})
	}
	if _, err := s.col.InsertMany(ctx, docs); err != nil {
		return nil, status.Errorf(codes.Internal, "insertMany: %v", err)
	}
	return &pb.Ack{Ok: true, Message: fmt.Sprintf("seeded %d tokens", len(docs))}, nil
}

func (s *server) NextToken(ctx context.Context, _ *pb.NextTokenRequest) (*pb.TokenReply, error) {
	// pop the oldest doc so identical RPCs return different tokens
	opts := options.FindOneAndDelete().SetSort(bson.D{{Key: "_id", Value: 1}})
	var out bson.M
	if err := s.col.FindOneAndDelete(ctx, bson.D{}, opts).Decode(&out); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "no tokens left")
		}
		return nil, status.Errorf(codes.Internal, "findOneAndDelete: %v", err)
	}
	token, _ := out["token"].(string)
	return &pb.TokenReply{Token: token}, nil
}

func main() {
	addr := ":50051"
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongoConnect(ctx, mongoURI)
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	defer client.Disconnect(context.Background())

	col := client.Database("keploydb").Collection("tokens")
	s := &server{col: col}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTokenServiceServer(grpcServer, s)

	log.Printf("gRPC server on %s (mongo: %s)", addr, mongoURI)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
