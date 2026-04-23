// Package store defines data models for the load-test MySQL API.
package store

import "time"

// Customer represents a registered customer.
type Customer struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Segment   string    `json:"segment"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateCustomerRequest is the request body for POST /customers.
type CreateCustomerRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Segment  string `json:"segment"`
}

// Product represents a purchasable product.
type Product struct {
	ID             string    `json:"id"`
	SKU            string    `json:"sku"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	PriceCents     int       `json:"price_cents"`
	InventoryCount int       `json:"inventory_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateProductRequest is the request body for POST /products.
type CreateProductRequest struct {
	SKU            string `json:"sku"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	PriceCents     int    `json:"price_cents"`
	InventoryCount int    `json:"inventory_count"`
}

// OrderItem is a line item within an order.
type OrderItem struct {
	ProductID      string `json:"product_id"`
	SKU            string `json:"sku"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
	LineTotalCents int    `json:"line_total_cents"`
}

// Order represents a customer order.
type Order struct {
	ID         string      `json:"id"`
	Customer   Customer    `json:"customer"`
	Status     string      `json:"status"`
	TotalCents int         `json:"total_cents"`
	CreatedAt  time.Time   `json:"created_at"`
	Items      []OrderItem `json:"items"`
}

// OrderItemInput is a single item in CreateOrderRequest.
type OrderItemInput struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// CreateOrderRequest is the request body for POST /orders.
type CreateOrderRequest struct {
	CustomerID string           `json:"customer_id"`
	Status     string           `json:"status"`
	Items      []OrderItemInput `json:"items"`
}

// OrderSearchParams holds query parameters for GET /orders.
type OrderSearchParams struct {
	Status         string
	CustomerID     string
	MinTotalCents  int
	CreatedFrom    *time.Time
	CreatedThrough *time.Time
	Limit          int
	Offset         int
}

// OrderSearchResult is a lightweight order row returned by GET /orders.
type OrderSearchResult struct {
	ID               string    `json:"id"`
	CustomerID       string    `json:"customer_id"`
	CustomerName     string    `json:"customer_name"`
	Status           string    `json:"status"`
	TotalCents       int       `json:"total_cents"`
	CreatedAt        time.Time `json:"created_at"`
	TotalItems       int       `json:"total_items"`
	DistinctProducts int       `json:"distinct_products"`
}

// CustomerSummary is the response for GET /customers/{id}/summary.
type CustomerSummary struct {
	Customer               Customer   `json:"customer"`
	OrdersCount            int        `json:"orders_count"`
	LifetimeValueCents     int        `json:"lifetime_value_cents"`
	AverageOrderValueCents int        `json:"average_order_value_cents"`
	FavoriteCategory       string     `json:"favorite_category"`
	LastOrderAt            *time.Time `json:"last_order_at,omitempty"`
}

// TopProduct is a single row in the GET /analytics/top-products response.
type TopProduct struct {
	ID           string `json:"id"`
	SKU          string `json:"sku"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	UnitsSold    int    `json:"units_sold"`
	RevenueCents int    `json:"revenue_cents"`
	OrdersCount  int    `json:"orders_count"`
	RevenueRank  int    `json:"revenue_rank"`
}

// LargePayloadRecord is the metadata-only view of a stored large payload.
type LargePayloadRecord struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ContentType      string    `json:"content_type"`
	PayloadSizeBytes int       `json:"payload_size_bytes"`
	SHA256           string    `json:"sha256"`
	CreatedAt        time.Time `json:"created_at"`
}

// LargePayloadDetail includes the actual payload bytes.
type LargePayloadDetail struct {
	LargePayloadRecord
	Payload string `json:"payload"`
}

// CreateLargePayloadRequest is the request body for POST /large-payloads.
type CreateLargePayloadRequest struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Payload     string `json:"payload"`
}

// DeleteLargePayloadResponse is the response body for DELETE /large-payloads/{id}.
type DeleteLargePayloadResponse struct {
	Deleted bool               `json:"deleted"`
	Record  LargePayloadRecord `json:"record"`
}
