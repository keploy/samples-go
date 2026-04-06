package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"

	pb "zepto-grpc/searchpb"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// buildFacetEntry builds a FacetEntry wire: { 1: name, 2: { FacetValue } }
func buildFacetEntry(name string, numericVal *float64, textVal *string) []byte {
	var fv []byte // FacetValue oneof
	if numericVal != nil {
		fv = protowire.AppendTag(fv, 2, protowire.Fixed64Type)
		fv = protowire.AppendFixed64(fv, math.Float64bits(*numericVal))
	} else if textVal != nil {
		fv = protowire.AppendTag(fv, 3, protowire.BytesType)
		fv = protowire.AppendString(fv, *textVal)
	}
	var entry []byte
	entry = protowire.AppendTag(entry, 1, protowire.BytesType)
	entry = protowire.AppendString(entry, name)
	entry = protowire.AppendTag(entry, 2, protowire.BytesType)
	entry = protowire.AppendBytes(entry, fv)
	return entry
}

// buildResponse constructs SearchResponse wire bytes with randomized
// repeated-field ordering inside the facet buckets.
func buildResponse() []byte {
	var buf []byte

	// field 1: float score = 67.0
	buf = protowire.AppendTag(buf, 1, protowire.Fixed32Type)
	buf = protowire.AppendFixed32(buf, math.Float32bits(67.0))

	// field 4: string hits_json
	hitsJSON := `{"hits":[{"_index":"pvid_search_products_v4","_score":15100000000000000000,` +
		`"_source":{"_rankingInfo":{"typosPresent":true,"numberOfWordsMatched":1}},` +
		`"match_type":"Other","attributes":{"subThemes":null},` +
		`"_id":"4f30407c-6a3c-4a4e-8a3d-652217d4b6cb_d67c25f8-3adb-40c1-9113-b46d54a6e8aa",` +
		`"trimming_meta":{"trimming_type":"L3"}}]}`
	buf = protowire.AppendTag(buf, 4, protowire.BytesType)
	buf = protowire.AppendString(buf, hitsJSON)

	// field 8: int32 total = 0
	buf = protowire.AppendTag(buf, 8, protowire.VarintType)
	buf = protowire.AppendVarint(buf, 0)

	// --- field 9: FacetInfo message ---

	// Build availability bucket entries (field 3 inside FacetInfo)
	zero := 0.0
	ovs := "OVS"
	availEntries := [][]byte{
		buildFacetEntry("candidateCnt", &zero, nil),
		buildFacetEntry("type", nil, &ovs),
	}
	// RANDOMIZE repeated entries — triggers the bug
	rand.Shuffle(len(availEntries), func(i, j int) {
		availEntries[i], availEntries[j] = availEntries[j], availEntries[i]
	})
	var availBucket []byte
	for _, e := range availEntries {
		availBucket = protowire.AppendTag(availBucket, 1, protowire.BytesType)
		availBucket = protowire.AppendBytes(availBucket, e)
	}

	// Build pricing bucket entries (field 2 inside FacetInfo)
	one := 1.0
	pricingEntries := [][]byte{
		buildFacetEntry("candidateCnt", &one, nil),
		buildFacetEntry("resultCnt", &one, nil),
	}
	rand.Shuffle(len(pricingEntries), func(i, j int) {
		pricingEntries[i], pricingEntries[j] = pricingEntries[j], pricingEntries[i]
	})
	var pricingBucket []byte
	for _, e := range pricingEntries {
		pricingBucket = protowire.AppendTag(pricingBucket, 1, protowire.BytesType)
		pricingBucket = protowire.AppendBytes(pricingBucket, e)
	}

	// Assemble FacetInfo: field 3 = availability, field 2 = pricing
	var facetInfo []byte
	facetInfo = protowire.AppendTag(facetInfo, 3, protowire.BytesType)
	facetInfo = protowire.AppendBytes(facetInfo, availBucket)
	facetInfo = protowire.AppendTag(facetInfo, 2, protowire.BytesType)
	facetInfo = protowire.AppendBytes(facetInfo, pricingBucket)

	buf = protowire.AppendTag(buf, 9, protowire.BytesType)
	buf = protowire.AppendBytes(buf, facetInfo)

	return buf
}

// rawCodec sends pre-built wire bytes without proto re-marshaling.
type rawCodec struct{}

func (rawCodec) Name() string { return "proto" }
func (rawCodec) Marshal(v interface{}) ([]byte, error) {
	if b, ok := v.(*rawFrame); ok {
		return b.data, nil
	}
	return proto.Marshal(v.(proto.Message))
}
func (rawCodec) Unmarshal(data []byte, v interface{}) error {
	if b, ok := v.(*rawFrame); ok {
		b.data = append(b.data[:0], data...)
		return nil
	}
	return proto.Unmarshal(data, v.(proto.Message))
}

type rawFrame struct{ data []byte }

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.ForceServerCodec(rawCodec{}))

	// Register service with a custom handler that returns raw wire bytes
	// with randomized field ordering each time.
	sd := grpc.ServiceDesc{
		ServiceName: "search.SearchService",
		HandlerType: (*pb.SearchServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Search",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					req := new(pb.SearchRequest)
					if err := dec(req); err != nil {
						return nil, err
					}
					log.Printf("Received search query: %s", req.GetQuery())
					return &rawFrame{data: buildResponse()}, nil
				},
			},
		},
	}
	s.RegisterService(&sd, &struct{ pb.UnimplementedSearchServiceServer }{})

	fmt.Println("gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
