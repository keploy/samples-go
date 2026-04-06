package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "zepto-grpc/searchpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewSearchServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Search(ctx, &pb.SearchRequest{Query: "shoes"})
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	fmt.Printf("Score:    %.1f\n", resp.GetScore())
	fmt.Printf("Total:    %d\n", resp.GetTotal())
	fmt.Printf("HitsJSON: %.80s...\n", resp.GetHitsJson())
	fmt.Println("Facets:")
	if f := resp.GetFacets(); f != nil {
		if a := f.GetAvailability(); a != nil {
			fmt.Println("  availability:")
			for _, e := range a.GetEntries() {
				fmt.Printf("    %s → ", e.GetName())
				if d := e.GetData(); d != nil {
					switch v := d.GetValue().(type) {
					case *pb.FacetValue_Numeric:
						fmt.Printf("%.1f\n", v.Numeric)
					case *pb.FacetValue_Text:
						fmt.Printf("%s\n", v.Text)
					}
				}
			}
		}
		if p := f.GetPricing(); p != nil {
			fmt.Println("  pricing:")
			for _, e := range p.GetEntries() {
				fmt.Printf("    %s → ", e.GetName())
				if d := e.GetData(); d != nil {
					switch v := d.GetValue().(type) {
					case *pb.FacetValue_Numeric:
						fmt.Printf("%.1f\n", v.Numeric)
					case *pb.FacetValue_Text:
						fmt.Printf("%s\n", v.Text)
					}
				}
			}
		}
	}
}
