package grpcapi

import (
	"context"
	"errors"
	"time"

	pb "loadtestgrpcapi/api/proto"
	"loadtestgrpcapi/internal/store"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements pb.LoadTestServiceServer backed by an in-memory store.
type Server struct {
	pb.UnimplementedLoadTestServiceServer
	store *store.Store
}

// New creates a new gRPC server with the given store.
func New(s *store.Store) *Server {
	return &Server{store: s}
}

// ─── Customer ────────────────────────────────────────────────────────────────

func (s *Server) CreateCustomer(_ context.Context, req *pb.CreateCustomerRequest) (*pb.Customer, error) {
	c, err := s.store.CreateCustomer(req.Email, req.FullName, req.Segment)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create customer: %v", err)
	}
	return customerPB(c), nil
}

func (s *Server) GetCustomerSummary(_ context.Context, req *pb.GetCustomerSummaryRequest) (*pb.CustomerSummary, error) {
	sum, err := s.store.GetCustomerSummary(req.CustomerId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "customer %s not found", req.CustomerId)
		}
		return nil, status.Errorf(codes.Internal, "get customer summary: %v", err)
	}
	lastOrder := ""
	if !sum.LastOrderAt.IsZero() {
		lastOrder = sum.LastOrderAt.Format(time.RFC3339)
	}
	return &pb.CustomerSummary{
		Customer:               customerPB(sum.Customer),
		OrdersCount:            sum.OrdersCount,
		LifetimeValueCents:     sum.LifetimeValueCents,
		AverageOrderValueCents: sum.AverageOrderValueCents,
		FavoriteCategory:       sum.FavoriteCategory,
		LastOrderAt:            lastOrder,
	}, nil
}

// ─── Product ─────────────────────────────────────────────────────────────────

func (s *Server) CreateProduct(_ context.Context, req *pb.CreateProductRequest) (*pb.Product, error) {
	p, err := s.store.CreateProduct(req.Sku, req.Name, req.Category, req.PriceCents, req.InventoryCount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create product: %v", err)
	}
	return productPB(p), nil
}

// ─── Order ───────────────────────────────────────────────────────────────────

func (s *Server) CreateOrder(_ context.Context, req *pb.CreateOrderRequest) (*pb.Order, error) {
	inputs := make([]store.OrderItemInput, len(req.Items))
	for i, it := range req.Items {
		inputs[i] = store.OrderItemInput{ProductID: it.ProductId, Quantity: it.Quantity}
	}
	o, err := s.store.CreateOrder(req.CustomerId, req.Status, inputs)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "resource not found: %v", err)
		}
		if errors.Is(err, store.ErrOutOfStock) {
			return nil, status.Errorf(codes.FailedPrecondition, "out of stock: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "create order: %v", err)
	}
	order, cust, err2 := s.store.GetOrder(o.ID)
	if err2 != nil {
		return nil, status.Errorf(codes.Internal, "get order after create: %v", err2)
	}
	return orderPB(order, cust), nil
}

func (s *Server) GetOrder(_ context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	o, c, err := s.store.GetOrder(req.OrderId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "order %s not found", req.OrderId)
		}
		return nil, status.Errorf(codes.Internal, "get order: %v", err)
	}
	return orderPB(o, c), nil
}

func (s *Server) SearchOrders(_ context.Context, req *pb.SearchOrdersRequest) (*pb.SearchOrdersResponse, error) {
	var from, through time.Time
	if req.CreatedFrom != "" {
		if t, err := time.Parse(time.RFC3339, req.CreatedFrom); err == nil {
			from = t
		}
	}
	if req.CreatedThrough != "" {
		if t, err := time.Parse(time.RFC3339, req.CreatedThrough); err == nil {
			through = t
		}
	}
	results, err := s.store.SearchOrders(req.Status, req.CustomerId, req.MinTotalCents, from, through, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search orders: %v", err)
	}
	pbResults := make([]*pb.OrderSearchResult, len(results))
	for i, r := range results {
		pbResults[i] = &pb.OrderSearchResult{
			OrderId:          r.OrderID,
			CustomerId:       r.CustomerID,
			CustomerName:     r.CustomerName,
			Status:           r.Status,
			TotalCents:       r.TotalCents,
			CreatedAt:        r.CreatedAt.Format(time.RFC3339),
			TotalItems:       r.TotalItems,
			DistinctProducts: r.DistinctProducts,
		}
	}
	return &pb.SearchOrdersResponse{Results: pbResults}, nil
}

// ─── Analytics ───────────────────────────────────────────────────────────────

func (s *Server) TopProducts(_ context.Context, req *pb.TopProductsRequest) (*pb.TopProductsResponse, error) {
	products, err := s.store.TopProducts(req.Days, req.Limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "top products: %v", err)
	}
	pbProducts := make([]*pb.TopProduct, len(products))
	for i, p := range products {
		pbProducts[i] = &pb.TopProduct{
			ProductId:    p.ProductID,
			Sku:          p.SKU,
			Name:         p.Name,
			Category:     p.Category,
			UnitsSold:    p.UnitsSold,
			RevenueCents: p.RevenueCents,
			OrdersCount:  p.OrdersCount,
			RevenueRank:  p.RevenueRank,
		}
	}
	return &pb.TopProductsResponse{Products: pbProducts}, nil
}

// ─── Large payloads ──────────────────────────────────────────────────────────

func (s *Server) CreateLargePayload(_ context.Context, req *pb.CreateLargePayloadRequest) (*pb.LargePayloadRecord, error) {
	lp, err := s.store.CreateLargePayload(req.Name, req.ContentType, req.Payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create large payload: %v", err)
	}
	return largePayloadRecordPB(lp), nil
}

func (s *Server) GetLargePayload(_ context.Context, req *pb.GetLargePayloadRequest) (*pb.LargePayloadDetail, error) {
	lp, err := s.store.GetLargePayload(req.PayloadId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "payload %s not found", req.PayloadId)
		}
		return nil, status.Errorf(codes.Internal, "get large payload: %v", err)
	}
	return &pb.LargePayloadDetail{
		Record:  largePayloadRecordPB(lp),
		Payload: lp.Payload,
	}, nil
}

func (s *Server) DeleteLargePayload(_ context.Context, req *pb.DeleteLargePayloadRequest) (*pb.DeleteLargePayloadResponse, error) {
	lp, err := s.store.DeleteLargePayload(req.PayloadId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "payload %s not found", req.PayloadId)
		}
		return nil, status.Errorf(codes.Internal, "delete large payload: %v", err)
	}
	return &pb.DeleteLargePayloadResponse{
		Deleted: true,
		Record:  largePayloadRecordPB(lp),
	}, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func customerPB(c *store.Customer) *pb.Customer {
	return &pb.Customer{
		Id:        c.ID,
		Email:     c.Email,
		FullName:  c.FullName,
		Segment:   c.Segment,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}
}

func productPB(p *store.Product) *pb.Product {
	return &pb.Product{
		Id:             p.ID,
		Sku:            p.SKU,
		Name:           p.Name,
		Category:       p.Category,
		PriceCents:     p.PriceCents,
		InventoryCount: p.InventoryCount,
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
	}
}

func orderPB(o *store.Order, c *store.Customer) *pb.Order {
	items := make([]*pb.OrderItem, len(o.Items))
	for i, it := range o.Items {
		items[i] = &pb.OrderItem{
			ProductId:      it.ProductID,
			Sku:            it.SKU,
			Name:           it.Name,
			Category:       it.Category,
			Quantity:       it.Quantity,
			UnitPriceCents: it.UnitPriceCents,
			LineTotalCents: it.LineTotalCents,
		}
	}
	var custPB *pb.Customer
	if c != nil {
		custPB = customerPB(c)
	}
	return &pb.Order{
		Id:         o.ID,
		Customer:   custPB,
		Status:     o.Status,
		TotalCents: o.TotalCents,
		CreatedAt:  o.CreatedAt.Format(time.RFC3339),
		Items:      items,
	}
}

func largePayloadRecordPB(lp *store.LargePayload) *pb.LargePayloadRecord {
	return &pb.LargePayloadRecord{
		Id:               lp.ID,
		Name:             lp.Name,
		ContentType:      lp.ContentType,
		PayloadSizeBytes: lp.PayloadSizeBytes,
		Sha256:           lp.SHA256,
		CreatedAt:        lp.CreatedAt.Format(time.RFC3339),
	}
}
