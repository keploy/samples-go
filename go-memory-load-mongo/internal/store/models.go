// Package store defines data models for the load-test MongoDB API.
package store

import "time"

type Customer struct {
	ID        string    `json:"id" bson:"_id,omitempty"`
	Email     string    `json:"email" bson:"email"`
	FullName  string    `json:"full_name" bson:"full_name"`
	Segment   string    `json:"segment" bson:"segment"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type Product struct {
	ID             string    `json:"id" bson:"_id,omitempty"`
	SKU            string    `json:"sku" bson:"sku"`
	Name           string    `json:"name" bson:"name"`
	Category       string    `json:"category" bson:"category"`
	PriceCents     int       `json:"price_cents" bson:"price_cents"`
	InventoryCount int       `json:"inventory_count" bson:"inventory_count"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
}

type OrderItemInput struct {
	ProductID string `json:"product_id" bson:"product_id"`
	Quantity  int    `json:"quantity" bson:"quantity"`
}

type OrderItem struct {
	ProductID      string `json:"product_id" bson:"product_id"`
	SKU            string `json:"sku" bson:"sku"`
	Name           string `json:"name" bson:"name"`
	Category       string `json:"category" bson:"category"`
	Quantity       int    `json:"quantity" bson:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents" bson:"unit_price_cents"`
	LineTotalCents int    `json:"line_total_cents" bson:"line_total_cents"`
}

type Order struct {
	ID         string      `json:"id" bson:"_id,omitempty"`
	Customer   Customer    `json:"customer" bson:"customer"`
	Status     string      `json:"status" bson:"status"`
	TotalCents int         `json:"total_cents" bson:"total_cents"`
	CreatedAt  time.Time   `json:"created_at" bson:"created_at"`
	Items      []OrderItem `json:"items" bson:"items"`
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
	CustomerID string           `json:"customer_id"`
	Status     string           `json:"status"`
	Items      []OrderItemInput `json:"items"`
}

type CreateLargePayloadRequest struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Payload     string `json:"payload"`
}

type LargePayloadRecord struct {
	ID               string    `json:"id" bson:"_id,omitempty"`
	Name             string    `json:"name" bson:"name"`
	ContentType      string    `json:"content_type" bson:"content_type"`
	PayloadSizeBytes int       `json:"payload_size_bytes" bson:"payload_size_bytes"`
	SHA256           string    `json:"sha256" bson:"sha256"`
	CreatedAt        time.Time `json:"created_at" bson:"created_at"`
}

type LargePayloadDetail struct {
	LargePayloadRecord `bson:",inline"`
	Payload            string `json:"payload" bson:"payload"`
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
	CustomerID       string    `json:"customer_id"`
	CustomerName     string    `json:"customer_name"`
	Status           string    `json:"status"`
	TotalCents       int       `json:"total_cents"`
	CreatedAt        time.Time `json:"created_at"`
	TotalItems       int       `json:"total_items"`
	DistinctProducts int       `json:"distinct_products"`
}

type OrderSearchParams struct {
	Status         string
	CustomerID     string
	MinTotalCents  int
	CreatedFrom    *time.Time
	CreatedThrough *time.Time
	Limit          int
	Offset         int
}

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
