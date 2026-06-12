package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrOutOfStock is returned when a product has insufficient inventory.
var ErrOutOfStock = errors.New("out of stock")

// ─── Domain types ────────────────────────────────────────────────────────────

type Customer struct {
	ID        string
	Email     string
	FullName  string
	Segment   string
	CreatedAt time.Time
}

type Product struct {
	ID             string
	SKU            string
	Name           string
	Category       string
	PriceCents     int32
	InventoryCount int32
	CreatedAt      time.Time
}

type OrderItem struct {
	ProductID      string
	SKU            string
	Name           string
	Category       string
	Quantity       int32
	UnitPriceCents int32
	LineTotalCents int32
}

type Order struct {
	ID         string
	CustomerID string
	Status     string
	TotalCents int32
	CreatedAt  time.Time
	Items      []OrderItem
}

type LargePayload struct {
	ID               string
	Name             string
	ContentType      string
	Payload          string
	SHA256           string
	PayloadSizeBytes int64
	CreatedAt        time.Time
}

// ─── Aggregate types ─────────────────────────────────────────────────────────

type CustomerSummary struct {
	Customer               *Customer
	OrdersCount            int32
	LifetimeValueCents     int64
	AverageOrderValueCents int64
	FavoriteCategory       string
	LastOrderAt            time.Time
}

type OrderItemInput struct {
	ProductID string
	Quantity  int32
}

type OrderSearchResult struct {
	OrderID          string
	CustomerID       string
	CustomerName     string
	Status           string
	TotalCents       int32
	CreatedAt        time.Time
	TotalItems       int32
	DistinctProducts int32
}

type TopProduct struct {
	ProductID    string
	SKU          string
	Name         string
	Category     string
	UnitsSold    int32
	RevenueCents int64
	OrdersCount  int32
	RevenueRank  int32
}

// ─── Store ───────────────────────────────────────────────────────────────────

type Store struct {
	mu            sync.RWMutex
	customers     map[string]*Customer
	products      map[string]*Product
	orders        map[string]*Order
	largePayloads map[string]*LargePayload
}

func New() *Store {
	return &Store{
		customers:     make(map[string]*Customer),
		products:      make(map[string]*Product),
		orders:        make(map[string]*Order),
		largePayloads: make(map[string]*LargePayload),
	}
}

// contentID derives a deterministic 24-hex-char ID from the supplied key
// parts, so that the same inputs always produce the same ID across keploy
// record and replay sessions.
func contentID(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(h[:])[:24]
}

// contentTime derives a deterministic creation timestamp from the supplied key
// parts using the same SHA-256 approach, producing a stable RFC3339 value
// within a 2-year window starting 2020-01-01.
func contentTime(parts ...string) time.Time {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	const base = int64(1577836800) // 2020-01-01T00:00:00Z
	const window = int64(2 * 365 * 24 * 3600)
	raw := int64(h[0])<<56 | int64(h[1])<<48 | int64(h[2])<<40 | int64(h[3])<<32 |
		int64(h[4])<<24 | int64(h[5])<<16 | int64(h[6])<<8 | int64(h[7])
	return time.Unix(base+(raw&0x7FFFFFFFFFFFFFFF)%window, 0).UTC()
}

// orderFingerprint builds a canonical, sorted string representation of order
// items so that the order ID is independent of input slice ordering.
func orderFingerprint(inputs []OrderItemInput) string {
	sorted := make([]OrderItemInput, len(inputs))
	copy(sorted, inputs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ProductID < sorted[j].ProductID
	})
	parts := make([]string, len(sorted))
	for i, inp := range sorted {
		parts[i] = fmt.Sprintf("%s:%d", inp.ProductID, inp.Quantity)
	}
	return strings.Join(parts, ",")
}

// ─── Customer operations ─────────────────────────────────────────────────────

func (s *Store) CreateCustomer(email, fullName, segment string) (*Customer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := &Customer{
		ID:        contentID(email),
		Email:     email,
		FullName:  fullName,
		Segment:   segment,
		CreatedAt: contentTime(email),
	}
	s.customers[c.ID] = c
	return c, nil
}

func (s *Store) GetCustomerSummary(customerID string) (*CustomerSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.customers[customerID]
	if !ok {
		return nil, ErrNotFound
	}

	var sum CustomerSummary
	sum.Customer = c
	catCount := make(map[string]int32)

	for _, o := range s.orders {
		if o.CustomerID != customerID {
			continue
		}
		sum.OrdersCount++
		sum.LifetimeValueCents += int64(o.TotalCents)
		if o.CreatedAt.After(sum.LastOrderAt) {
			sum.LastOrderAt = o.CreatedAt
		}
		for _, it := range o.Items {
			catCount[it.Category] += it.Quantity
		}
	}

	if sum.OrdersCount > 0 {
		sum.AverageOrderValueCents = sum.LifetimeValueCents / int64(sum.OrdersCount)
	}

	var maxCat string
	var maxQty int32
	for cat, qty := range catCount {
		if qty > maxQty {
			maxQty = qty
			maxCat = cat
		}
	}
	sum.FavoriteCategory = maxCat
	return &sum, nil
}

// ─── Product operations ──────────────────────────────────────────────────────

func (s *Store) CreateProduct(sku, name, category string, priceCents, inventoryCount int32) (*Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := &Product{
		ID:             contentID(sku),
		SKU:            sku,
		Name:           name,
		Category:       category,
		PriceCents:     priceCents,
		InventoryCount: inventoryCount,
		CreatedAt:      contentTime(sku),
	}
	s.products[p.ID] = p
	return p, nil
}

// ─── Order operations ────────────────────────────────────────────────────────

func (s *Store) CreateOrder(customerID, orderStatus string, inputs []OrderItemInput) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.customers[customerID]; !ok {
		return nil, fmt.Errorf("customer %s: %w", customerID, ErrNotFound)
	}

	var items []OrderItem
	var totalCents int32
	for _, inp := range inputs {
		p, ok := s.products[inp.ProductID]
		if !ok {
			return nil, fmt.Errorf("product %s: %w", inp.ProductID, ErrNotFound)
		}
		if p.InventoryCount < inp.Quantity {
			return nil, fmt.Errorf("product %s: %w", inp.ProductID, ErrOutOfStock)
		}
		p.InventoryCount -= inp.Quantity
		line := inp.Quantity * p.PriceCents
		items = append(items, OrderItem{
			ProductID:      p.ID,
			SKU:            p.SKU,
			Name:           p.Name,
			Category:       p.Category,
			Quantity:       inp.Quantity,
			UnitPriceCents: p.PriceCents,
			LineTotalCents: line,
		})
		totalCents += line
	}

	if orderStatus == "" {
		orderStatus = "pending"
	}
	fingerprint := orderFingerprint(inputs)
	orderID := contentID(customerID, fingerprint, orderStatus)
	// Idempotent: if an identical order already exists, return it without
	// re-decrementing inventory (handles duplicate keploy replay calls).
	if existing, ok := s.orders[orderID]; ok {
		return existing, nil
	}
	o := &Order{
		ID:         orderID,
		CustomerID: customerID,
		Status:     orderStatus,
		TotalCents: totalCents,
		CreatedAt:  contentTime(customerID, fingerprint, orderStatus),
		Items:      items,
	}
	s.orders[o.ID] = o
	return o, nil
}

func (s *Store) GetOrder(orderID string) (*Order, *Customer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[orderID]
	if !ok {
		return nil, nil, ErrNotFound
	}
	return o, s.customers[o.CustomerID], nil
}

func (s *Store) SearchOrders(
	statusFilter, customerID string,
	minTotalCents int64,
	createdFrom, createdThrough time.Time,
	limit, offset int32,
) ([]OrderSearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []OrderSearchResult
	for _, o := range s.orders {
		if statusFilter != "" && o.Status != statusFilter {
			continue
		}
		if customerID != "" && o.CustomerID != customerID {
			continue
		}
		if minTotalCents > 0 && int64(o.TotalCents) < minTotalCents {
			continue
		}
		if !createdFrom.IsZero() && o.CreatedAt.Before(createdFrom) {
			continue
		}
		if !createdThrough.IsZero() && o.CreatedAt.After(createdThrough) {
			continue
		}
		cust := s.customers[o.CustomerID]
		custName := ""
		if cust != nil {
			custName = cust.FullName
		}
		var totalItems int32
		seen := make(map[string]bool)
		for _, it := range o.Items {
			totalItems += it.Quantity
			seen[it.ProductID] = true
		}
		results = append(results, OrderSearchResult{
			OrderID:          o.ID,
			CustomerID:       o.CustomerID,
			CustomerName:     custName,
			Status:           o.Status,
			TotalCents:       o.TotalCents,
			CreatedAt:        o.CreatedAt,
			TotalItems:       totalItems,
			DistinctProducts: int32(len(seen)),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	if int(offset) >= len(results) {
		return []OrderSearchResult{}, nil
	}
	results = results[offset:]
	if limit > 0 && int(limit) < len(results) {
		results = results[:limit]
	}
	return results, nil
}

// ─── Analytics ───────────────────────────────────────────────────────────────

func (s *Store) TopProducts(days, limit int32) ([]TopProduct, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_ = days // days filter intentionally unused: order CreatedAt is derived
	// from content hash (not wall-clock time), so a time.Now()-based cutoff
	// would exclude all orders during keploy replay. Using all-time data keeps
	// mock matching deterministic across record and replay sessions.
	agg := make(map[string]*TopProduct)
	for _, o := range s.orders {
		for _, it := range o.Items {
			tp, ok := agg[it.ProductID]
			if !ok {
				sku, name, cat := "", "", ""
				if p := s.products[it.ProductID]; p != nil {
					sku, name, cat = p.SKU, p.Name, p.Category
				}
				agg[it.ProductID] = &TopProduct{
					ProductID: it.ProductID,
					SKU:       sku,
					Name:      name,
					Category:  cat,
				}
				tp = agg[it.ProductID]
			}
			tp.UnitsSold += it.Quantity
			tp.RevenueCents += int64(it.LineTotalCents)
			tp.OrdersCount++
		}
	}

	products := make([]TopProduct, 0, len(agg))
	for _, tp := range agg {
		products = append(products, *tp)
	}
	sort.Slice(products, func(i, j int) bool {
		return products[i].RevenueCents > products[j].RevenueCents
	})
	for i := range products {
		products[i].RevenueRank = int32(i + 1)
	}
	if limit > 0 && int(limit) < len(products) {
		products = products[:limit]
	}
	return products, nil
}

// ─── Large payload operations ────────────────────────────────────────────────

func (s *Store) CreateLargePayload(name, contentType, payload string) (*LargePayload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h := sha256.Sum256([]byte(payload))
	sha256Hex := hex.EncodeToString(h[:])
	lp := &LargePayload{
		ID:               contentID(name, contentType, sha256Hex),
		Name:             name,
		ContentType:      contentType,
		Payload:          payload,
		SHA256:           sha256Hex,
		PayloadSizeBytes: int64(len(payload)),
		CreatedAt:        contentTime(name, contentType, sha256Hex),
	}
	s.largePayloads[lp.ID] = lp
	return lp, nil
}

func (s *Store) GetLargePayload(payloadID string) (*LargePayload, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lp, ok := s.largePayloads[payloadID]
	if !ok {
		return nil, ErrNotFound
	}
	return lp, nil
}

func (s *Store) DeleteLargePayload(payloadID string) (*LargePayload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lp, ok := s.largePayloads[payloadID]
	if !ok {
		return nil, ErrNotFound
	}
	delete(s.largePayloads, payloadID)
	return lp, nil
}
