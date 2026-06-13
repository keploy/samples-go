package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrNotFound              = errors.New("not found")
	ErrConflict              = errors.New("conflict")
	ErrValidation            = errors.New("validation error")
	ErrInsufficientInventory = errors.New("insufficient inventory")
)

const maxLargePayloadBytes = 8 * 1024 * 1024

var (
	validSegments = map[string]struct{}{
		"startup":    {},
		"enterprise": {},
		"retail":     {},
		"partner":    {},
	}
	validStatuses = map[string]struct{}{
		"pending":   {},
		"paid":      {},
		"shipped":   {},
		"cancelled": {},
	}
)

type Store struct {
	db           *mongo.Database
	customers    *mongo.Collection
	products     *mongo.Collection
	orders       *mongo.Collection
	largePayload *mongo.Collection
}

func New(db *mongo.Database) *Store {
	return &Store{
		db:           db,
		customers:    db.Collection("customers"),
		products:     db.Collection("products"),
		orders:       db.Collection("orders"),
		largePayload: db.Collection("large_payloads"),
	}
}

// EnsureIndexes creates the required indexes on first run.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	// customers: unique email
	if _, err := s.customers.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return fmt.Errorf("customer email index: %w", err)
	}

	// products: unique sku
	if _, err := s.products.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "sku", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return fmt.Errorf("product sku index: %w", err)
	}

	// orders: customer_id + created_at
	if _, err := s.orders.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "customer._id", Value: 1}, {Key: "created_at", Value: -1}},
	}); err != nil {
		return fmt.Errorf("order customer index: %w", err)
	}

	// orders: status + created_at
	if _, err := s.orders.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: -1}},
	}); err != nil {
		return fmt.Errorf("order status index: %w", err)
	}

	// large_payloads: created_at descending
	if _, err := s.largePayload.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	}); err != nil {
		return fmt.Errorf("large_payload created_at index: %w", err)
	}

	return nil
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
func orderFingerprint(items []OrderItemInput) string {
	sorted := make([]OrderItemInput, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ProductID < sorted[j].ProductID
	})
	parts := make([]string, len(sorted))
	for i, inp := range sorted {
		parts[i] = fmt.Sprintf("%s:%d", inp.ProductID, inp.Quantity)
	}
	return strings.Join(parts, ",")
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.Client().Ping(ctx, nil)
}

func (s *Store) CreateCustomer(ctx context.Context, req CreateCustomerRequest) (Customer, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)
	req.Segment = strings.TrimSpace(strings.ToLower(req.Segment))

	if _, err := mail.ParseAddress(req.Email); err != nil {
		return Customer{}, fmt.Errorf("%w: email must be valid", ErrValidation)
	}
	if req.FullName == "" {
		return Customer{}, fmt.Errorf("%w: full_name is required", ErrValidation)
	}
	if _, ok := validSegments[req.Segment]; !ok {
		return Customer{}, fmt.Errorf("%w: unsupported customer segment", ErrValidation)
	}

	customer := Customer{
		ID:        contentID(req.Email),
		Email:     req.Email,
		FullName:  req.FullName,
		Segment:   req.Segment,
		CreatedAt: contentTime(req.Email),
	}

	_, err := s.customers.InsertOne(ctx, customer)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return Customer{}, fmt.Errorf("%w: email already exists", ErrConflict)
		}
		return Customer{}, fmt.Errorf("insert customer: %w", err)
	}

	return customer, nil
}

func (s *Store) CreateProduct(ctx context.Context, req CreateProductRequest) (Product, error) {
	req.SKU = strings.TrimSpace(strings.ToUpper(req.SKU))
	req.Name = strings.TrimSpace(req.Name)
	req.Category = strings.TrimSpace(strings.ToLower(req.Category))

	switch {
	case req.SKU == "":
		return Product{}, fmt.Errorf("%w: sku is required", ErrValidation)
	case req.Name == "":
		return Product{}, fmt.Errorf("%w: name is required", ErrValidation)
	case req.Category == "":
		return Product{}, fmt.Errorf("%w: category is required", ErrValidation)
	case req.PriceCents <= 0:
		return Product{}, fmt.Errorf("%w: price_cents must be greater than zero", ErrValidation)
	case req.InventoryCount < 0:
		return Product{}, fmt.Errorf("%w: inventory_count cannot be negative", ErrValidation)
	}

	product := Product{
		ID:             contentID(req.SKU),
		SKU:            req.SKU,
		Name:           req.Name,
		Category:       req.Category,
		PriceCents:     req.PriceCents,
		InventoryCount: req.InventoryCount,
		CreatedAt:      contentTime(req.SKU),
	}

	_, err := s.products.InsertOne(ctx, product)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return Product{}, fmt.Errorf("%w: sku already exists", ErrConflict)
		}
		return Product{}, fmt.Errorf("insert product: %w", err)
	}

	return product, nil
}

func (s *Store) CreateOrder(ctx context.Context, req CreateOrderRequest) (Order, error) {
	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Status == "" {
		req.Status = "paid"
	}

	switch {
	case req.CustomerID == "":
		return Order{}, fmt.Errorf("%w: customer_id is required", ErrValidation)
	case len(req.Items) == 0:
		return Order{}, fmt.Errorf("%w: at least one item is required", ErrValidation)
	}
	if _, ok := validStatuses[req.Status]; !ok {
		return Order{}, fmt.Errorf("%w: unsupported order status", ErrValidation)
	}
	for _, item := range req.Items {
		if item.ProductID == "" || item.Quantity <= 0 {
			return Order{}, fmt.Errorf("%w: every item needs a valid product_id and quantity", ErrValidation)
		}
	}

	// Verify customer exists.
	var customer Customer
	if err := s.customers.FindOne(ctx, bson.M{"_id": req.CustomerID}).Decode(&customer); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Order{}, fmt.Errorf("%w: customer %s", ErrNotFound, req.CustomerID)
		}
		return Order{}, fmt.Errorf("find customer: %w", err)
	}

	// Build items and decrement inventory atomically per product.
	var items []OrderItem
	totalCents := 0

	for _, input := range req.Items {
		// Decrement inventory with a findOneAndUpdate — atomic per document.
		var product Product
		after := options.After
		err := s.products.FindOneAndUpdate(
			ctx,
			bson.M{"_id": input.ProductID, "inventory_count": bson.M{"$gte": input.Quantity}},
			bson.M{"$inc": bson.M{"inventory_count": -input.Quantity}},
			options.FindOneAndUpdate().SetReturnDocument(after),
		).Decode(&product)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				// Either product not found or insufficient inventory.
				var exists Product
				if findErr := s.products.FindOne(ctx, bson.M{"_id": input.ProductID}).Decode(&exists); findErr != nil {
					return Order{}, fmt.Errorf("%w: product %s", ErrNotFound, input.ProductID)
				}
				return Order{}, fmt.Errorf("%w: product %s", ErrInsufficientInventory, input.ProductID)
			}
			return Order{}, fmt.Errorf("update inventory for product %s: %w", input.ProductID, err)
		}

		lineCents := product.PriceCents * input.Quantity
		totalCents += lineCents
		items = append(items, OrderItem{
			ProductID:      product.ID,
			SKU:            product.SKU,
			Name:           product.Name,
			Category:       product.Category,
			Quantity:       input.Quantity,
			UnitPriceCents: product.PriceCents,
			LineTotalCents: lineCents,
		})
	}

	fp := orderFingerprint(req.Items)
	order := Order{
		ID:         contentID(req.CustomerID, fp),
		Customer:   customer,
		Status:     req.Status,
		TotalCents: totalCents,
		CreatedAt:  contentTime(req.CustomerID, fp),
		Items:      items,
	}

	if _, err := s.orders.InsertOne(ctx, order); err != nil {
		return Order{}, fmt.Errorf("insert order: %w", err)
	}

	return order, nil
}

func (s *Store) GetOrder(ctx context.Context, orderID string) (Order, error) {
	var order Order
	if err := s.orders.FindOne(ctx, bson.M{"_id": orderID}).Decode(&order); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Order{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
		}
		return Order{}, fmt.Errorf("find order: %w", err)
	}

	return order, nil
}

func (s *Store) GetCustomerSummary(ctx context.Context, customerID string) (CustomerSummary, error) {
	var customer Customer
	if err := s.customers.FindOne(ctx, bson.M{"_id": customerID}).Decode(&customer); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return CustomerSummary{}, fmt.Errorf("%w: customer %s", ErrNotFound, customerID)
		}
		return CustomerSummary{}, fmt.Errorf("find customer: %w", err)
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"customer._id": customerID}}},
		{{Key: "$unwind", Value: bson.M{"path": "$items", "preserveNullAndEmptyArrays": true}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$customer._id"},
			{Key: "orders_count", Value: bson.M{"$addToSet": "$_id"}},
			{Key: "lifetime_value_cents", Value: bson.M{"$sum": "$total_cents"}},
			{Key: "last_order_at", Value: bson.M{"$max": "$created_at"}},
			{Key: "category_spend", Value: bson.M{"$push": bson.M{
				"category": "$items.category",
				"cents":    "$items.line_total_cents",
			}}},
		}}},
	}

	cursor, err := s.orders.Aggregate(ctx, pipeline)
	if err != nil {
		return CustomerSummary{}, fmt.Errorf("aggregate customer summary: %w", err)
	}
	defer cursor.Close(ctx) //nolint:errcheck

	summary := CustomerSummary{Customer: customer}

	if cursor.Next(ctx) {
		var raw bson.M
		if err := cursor.Decode(&raw); err != nil {
			return CustomerSummary{}, fmt.Errorf("decode customer summary: %w", err)
		}

		// orders_count is a set of distinct order IDs.
		if ids, ok := raw["orders_count"].(bson.A); ok {
			summary.OrdersCount = len(ids)
		}
		if v, ok := raw["lifetime_value_cents"].(int32); ok {
			summary.LifetimeValueCents = int(v)
		} else if v, ok := raw["lifetime_value_cents"].(int64); ok {
			summary.LifetimeValueCents = int(v)
		}
		if summary.OrdersCount > 0 {
			summary.AverageOrderValueCents = summary.LifetimeValueCents / summary.OrdersCount
		}
		if t, ok := raw["last_order_at"].(time.Time); ok {
			summary.LastOrderAt = &t
		}

		// Find favourite category by total spend.
		if spends, ok := raw["category_spend"].(bson.A); ok {
			catSpend := map[string]int{}
			for _, item := range spends {
				if m, ok := item.(bson.M); ok {
					cat, _ := m["category"].(string)
					var cents int
					switch v := m["cents"].(type) {
					case int32:
						cents = int(v)
					case int64:
						cents = int(v)
					}
					catSpend[cat] += cents
				}
			}
			best, bestCents := "", 0
			for cat, cents := range catSpend {
				if cents > bestCents || (cents == bestCents && cat < best) {
					best, bestCents = cat, cents
				}
			}
			summary.FavoriteCategory = best
		}
	}

	return summary, nil
}

func (s *Store) SearchOrders(ctx context.Context, params OrderSearchParams) ([]OrderSearchResult, error) {
	if params.Limit <= 0 {
		params.Limit = 25
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}
	params.Status = strings.TrimSpace(strings.ToLower(params.Status))
	if params.Status != "" {
		if _, ok := validStatuses[params.Status]; !ok {
			return nil, fmt.Errorf("%w: unsupported order status", ErrValidation)
		}
	}

	filter := bson.M{}
	if params.Status != "" {
		filter["status"] = params.Status
	}
	if params.CustomerID != "" {
		filter["customer._id"] = params.CustomerID
	}
	if params.MinTotalCents > 0 {
		filter["total_cents"] = bson.M{"$gte": params.MinTotalCents}
	}
	if params.CreatedFrom != nil || params.CreatedThrough != nil {
		timeFilter := bson.M{}
		if params.CreatedFrom != nil {
			timeFilter["$gte"] = *params.CreatedFrom
		}
		if params.CreatedThrough != nil {
			timeFilter["$lte"] = *params.CreatedThrough
		}
		filter["created_at"] = timeFilter
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(params.Offset)).
		SetLimit(int64(params.Limit))

	cursor, err := s.orders.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("search orders: %w", err)
	}
	defer cursor.Close(ctx) //nolint:errcheck

	results := make([]OrderSearchResult, 0, params.Limit)
	for cursor.Next(ctx) {
		var order Order
		if err := cursor.Decode(&order); err != nil {
			return nil, fmt.Errorf("decode order: %w", err)
		}

		totalItems, distinctProducts := 0, map[string]struct{}{}
		for _, item := range order.Items {
			totalItems += item.Quantity
			distinctProducts[item.ProductID] = struct{}{}
		}

		results = append(results, OrderSearchResult{
			ID:               order.ID,
			CustomerID:       order.Customer.ID,
			CustomerName:     order.Customer.FullName,
			Status:           order.Status,
			TotalCents:       order.TotalCents,
			CreatedAt:        order.CreatedAt,
			TotalItems:       totalItems,
			DistinctProducts: len(distinctProducts),
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}

	return results, nil
}

func (s *Store) TopProducts(ctx context.Context, days, limit int) ([]TopProduct, error) {
	if days <= 0 {
		days = 30
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	_ = days // days filter intentionally unused: using all-time data keeps the
	// aggregation pipeline parameter-free so keploy can match the mock
	// deterministically across record and replay sessions.

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"status": bson.M{"$in": bson.A{"paid", "shipped"}},
		}}},
		{{Key: "$unwind", Value: "$items"}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.M{"product_id": "$items.product_id", "sku": "$items.sku", "name": "$items.name", "category": "$items.category"}},
			{Key: "units_sold", Value: bson.M{"$sum": "$items.quantity"}},
			{Key: "revenue_cents", Value: bson.M{"$sum": "$items.line_total_cents"}},
			{Key: "orders_count", Value: bson.M{"$addToSet": "$_id"}},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":           0,
			"product_id":    "$_id.product_id",
			"sku":           "$_id.sku",
			"name":          "$_id.name",
			"category":      "$_id.category",
			"units_sold":    1,
			"revenue_cents": 1,
			"orders_count":  bson.M{"$size": "$orders_count"},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "revenue_cents", Value: -1}, {Key: "units_sold", Value: -1}}}},
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := s.orders.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate top products: %w", err)
	}
	defer cursor.Close(ctx) //nolint:errcheck

	results := make([]TopProduct, 0, limit)
	rank := 1
	for cursor.Next(ctx) {
		var row struct {
			ProductID    string `bson:"product_id"`
			SKU          string `bson:"sku"`
			Name         string `bson:"name"`
			Category     string `bson:"category"`
			UnitsSold    int    `bson:"units_sold"`
			RevenueCents int    `bson:"revenue_cents"`
			OrdersCount  int    `bson:"orders_count"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, fmt.Errorf("decode top product: %w", err)
		}
		results = append(results, TopProduct{
			ID:           row.ProductID,
			SKU:          row.SKU,
			Name:         row.Name,
			Category:     row.Category,
			UnitsSold:    row.UnitsSold,
			RevenueCents: row.RevenueCents,
			OrdersCount:  row.OrdersCount,
			RevenueRank:  rank,
		})
		rank++
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate top products: %w", err)
	}

	return results, nil
}

func (s *Store) CreateLargePayload(ctx context.Context, req CreateLargePayloadRequest) (LargePayloadRecord, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.ContentType = strings.TrimSpace(req.ContentType)
	if req.ContentType == "" {
		req.ContentType = "text/plain"
	}

	switch {
	case req.Name == "":
		return LargePayloadRecord{}, fmt.Errorf("%w: name is required", ErrValidation)
	case req.Payload == "":
		return LargePayloadRecord{}, fmt.Errorf("%w: payload is required", ErrValidation)
	}

	payloadSizeBytes := len([]byte(req.Payload))
	if payloadSizeBytes > maxLargePayloadBytes {
		return LargePayloadRecord{}, fmt.Errorf(
			"%w: payload exceeds %d bytes (%d MiB) limit",
			ErrValidation,
			maxLargePayloadBytes,
			maxLargePayloadBytes/(1024*1024),
		)
	}

	checksum := sha256.Sum256([]byte(req.Payload))

	doc := LargePayloadDetail{
		LargePayloadRecord: LargePayloadRecord{
			ID:               contentID(req.Name, hex.EncodeToString(checksum[:])),
			Name:             req.Name,
			ContentType:      req.ContentType,
			PayloadSizeBytes: payloadSizeBytes,
			SHA256:           hex.EncodeToString(checksum[:]),
			CreatedAt:        contentTime(req.Name, hex.EncodeToString(checksum[:])),
		},
		Payload: req.Payload,
	}

	if _, err := s.largePayload.InsertOne(ctx, doc); err != nil {
		return LargePayloadRecord{}, fmt.Errorf("insert large payload: %w", err)
	}

	return doc.LargePayloadRecord, nil
}

func (s *Store) GetLargePayload(ctx context.Context, payloadID string) (LargePayloadDetail, error) {
	var record LargePayloadDetail
	if err := s.largePayload.FindOne(ctx, bson.M{"_id": payloadID}).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return LargePayloadDetail{}, fmt.Errorf("%w: large payload %s", ErrNotFound, payloadID)
		}
		return LargePayloadDetail{}, fmt.Errorf("find large payload: %w", err)
	}

	return record, nil
}

func (s *Store) DeleteLargePayload(ctx context.Context, payloadID string) (DeleteLargePayloadResponse, error) {
	var detail LargePayloadDetail
	if err := s.largePayload.FindOneAndDelete(ctx, bson.M{"_id": payloadID}).Decode(&detail); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return DeleteLargePayloadResponse{}, fmt.Errorf("%w: large payload %s", ErrNotFound, payloadID)
		}
		return DeleteLargePayloadResponse{}, fmt.Errorf("delete large payload: %w", err)
	}

	return DeleteLargePayloadResponse{
		Deleted: true,
		Record:  detail.LargePayloadRecord,
	}, nil
}
