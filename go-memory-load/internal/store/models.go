// Package store defines data models for the load-test API.
package store

import "time"

type Customer struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Segment   string    `json:"segment"`
	CreatedAt time.Time `json:"created_at"`
}

type Product struct {
	ID             int64     `json:"id"`
	SKU            string    `json:"sku"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	PriceCents     int       `json:"price_cents"`
	InventoryCount int       `json:"inventory_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type OrderItemInput struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type OrderItem struct {
	ProductID      int64  `json:"product_id"`
	SKU            string `json:"sku"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
	LineTotalCents int    `json:"line_total_cents"`
}

type Order struct {
	ID         string      `json:"id"`
	Customer   Customer    `json:"customer"`
	Status     string      `json:"status"`
	TotalCents int         `json:"total_cents"`
	CreatedAt  time.Time   `json:"created_at"`
	Items      []OrderItem `json:"items"`
}

type CreateCustomerRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Segment  string `json:"segment"`
}

type CreateProductRequest struct {
	SKU            string `json:"sku"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	PriceCents     int    `json:"price_cents"`
	InventoryCount int    `json:"inventory_count"`
}

type CreateOrderRequest struct {
	CustomerID int64            `json:"customer_id"`
	Status     string           `json:"status"`
	Items      []OrderItemInput `json:"items"`
}

type CreateLargePayloadRequest struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Payload     string `json:"payload"`
}

type LargePayloadRecord struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ContentType      string    `json:"content_type"`
	PayloadSizeBytes int       `json:"payload_size_bytes"`
	SHA256           string    `json:"sha256"`
	CreatedAt        time.Time `json:"created_at"`
}

type LargePayloadDetail struct {
	LargePayloadRecord
	Payload string `json:"payload"`
}

type DeleteLargePayloadResponse struct {
	Deleted bool               `json:"deleted"`
	Record  LargePayloadRecord `json:"record"`
}

type CustomerSummary struct {
	Customer               Customer   `json:"customer"`
	OrdersCount            int        `json:"orders_count"`
	LifetimeValueCents     int        `json:"lifetime_value_cents"`
	AverageOrderValueCents int        `json:"average_order_value_cents"`
	LastOrderAt            *time.Time `json:"last_order_at,omitempty"`
	FavoriteCategory       string     `json:"favorite_category,omitempty"`
}

type OrderSearchResult struct {
	ID               string    `json:"id"`
	CustomerID       int64     `json:"customer_id"`
	CustomerName     string    `json:"customer_name"`
	Status           string    `json:"status"`
	TotalCents       int       `json:"total_cents"`
	CreatedAt        time.Time `json:"created_at"`
	TotalItems       int       `json:"total_items"`
	DistinctProducts int       `json:"distinct_products"`
}

type OrderSearchParams struct {
	Status         string
	CustomerID     int64
	MinTotalCents  int
	CreatedFrom    *time.Time
	CreatedThrough *time.Time
	Limit          int
	Offset         int
}

type TopProduct struct {
	ID           int64  `json:"id"`
	SKU          string `json:"sku"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	UnitsSold    int    `json:"units_sold"`
	RevenueCents int    `json:"revenue_cents"`
	OrdersCount  int    `json:"orders_count"`
	RevenueRank  int    `json:"revenue_rank"`
}
