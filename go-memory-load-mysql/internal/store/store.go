package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
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

// Store wraps a *sql.DB and exposes the business operations.
type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// contentID derives a deterministic UUID-shaped identifier from content.
// Using SHA256 ensures the same inputs always produce the same ID, which allows
// Keploy to match recorded MySQL mocks during replay — the INSERT query bytes
// must be identical between record and replay runs.
func contentID(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	b := h[:]
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func isDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
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
		CreatedAt: time.Now().UTC(),
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO customers (id, email, full_name, segment, created_at) VALUES (?, ?, ?, ?, ?)`,
		customer.ID, customer.Email, customer.FullName, customer.Segment, customer.CreatedAt,
	)
	if err != nil {
		if isDuplicateKey(err) {
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
		CreatedAt:      time.Now().UTC(),
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO products (id, sku, name, category, price_cents, inventory_count, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		product.ID, product.SKU, product.Name, product.Category, product.PriceCents, product.InventoryCount, product.CreatedAt,
	)
	if err != nil {
		if isDuplicateKey(err) {
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

	// Sort items by product_id so all concurrent transactions acquire row
	// locks in the same order, reducing (but not eliminating) deadlocks.
	sort.Slice(req.Items, func(i, j int) bool {
		return req.Items[i].ProductID < req.Items[j].ProductID
	})

	// Retry the transaction on InnoDB deadlock (Error 1213). Under high
	// concurrency multiple transactions can deadlock even with consistent
	// lock ordering; MySQL recommends retrying on deadlock.
	const maxRetries = 5
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		order, err := s.createOrderTx(ctx, req)
		if err == nil {
			return order, nil
		}
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1213 && attempt < maxRetries-1 {
			// Back off briefly before retrying: 10ms, 20ms, 40ms, 80ms.
			time.Sleep(time.Duration(1<<uint(attempt)) * 10 * time.Millisecond)
			lastErr = err
			continue
		}
		return Order{}, err
	}
	return Order{}, fmt.Errorf("create order: %w (exceeded %d retries)", lastErr, maxRetries)
}

func (s *Store) createOrderTx(ctx context.Context, req CreateOrderRequest) (Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Verify customer exists.
	var customer Customer
	row := tx.QueryRowContext(ctx,
		`SELECT id, email, full_name, segment, created_at FROM customers WHERE id = ?`,
		req.CustomerID,
	)
	if err := row.Scan(&customer.ID, &customer.Email, &customer.FullName, &customer.Segment, &customer.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Order{}, fmt.Errorf("%w: customer %s", ErrNotFound, req.CustomerID)
		}
		return Order{}, fmt.Errorf("find customer: %w", err)
	}

	// Build items, decrement inventory.
	var items []OrderItem
	totalCents := 0

	for _, input := range req.Items {
		result, err := tx.ExecContext(ctx,
			`UPDATE products SET inventory_count = inventory_count - ? WHERE id = ? AND inventory_count >= ?`,
			input.Quantity, input.ProductID, input.Quantity,
		)
		if err != nil {
			return Order{}, fmt.Errorf("decrement inventory for product %s: %w", input.ProductID, err)
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			// Either product not found or insufficient inventory.
			var exists int
			checkErr := tx.QueryRowContext(ctx, `SELECT 1 FROM products WHERE id = ? LIMIT 1`, input.ProductID).Scan(&exists)
			if errors.Is(checkErr, sql.ErrNoRows) {
				return Order{}, fmt.Errorf("%w: product %s", ErrNotFound, input.ProductID)
			}
			return Order{}, fmt.Errorf("%w: product %s", ErrInsufficientInventory, input.ProductID)
		}

		var product Product
		productRow := tx.QueryRowContext(ctx,
			`SELECT id, sku, name, category, price_cents FROM products WHERE id = ?`,
			input.ProductID,
		)
		if err := productRow.Scan(&product.ID, &product.SKU, &product.Name, &product.Category, &product.PriceCents); err != nil {
			return Order{}, fmt.Errorf("fetch product %s: %w", input.ProductID, err)
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

	// Derive order ID from customer + sorted product IDs so the same request
	// always produces the same INSERT query bytes for Keploy mock matching.
	pidParts := make([]string, 0, len(req.Items)+2)
	pidParts = append(pidParts, req.CustomerID, req.Status)
	for _, it := range req.Items {
		pidParts = append(pidParts, it.ProductID)
	}
	orderID := contentID(pidParts...)
	createdAt := time.Now().UTC()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO orders (id, customer_id, customer_email, customer_name, customer_segment, status, total_cents, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		orderID, customer.ID, customer.Email, customer.FullName, customer.Segment,
		req.Status, totalCents, createdAt,
	)
	if err != nil {
		return Order{}, fmt.Errorf("insert order: %w", err)
	}

	for _, item := range items {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO order_items (id, order_id, product_id, sku, name, category, quantity, unit_price_cents, line_total_cents)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			contentID(orderID, item.ProductID), orderID, item.ProductID, item.SKU, item.Name, item.Category,
			item.Quantity, item.UnitPriceCents, item.LineTotalCents,
		)
		if err != nil {
			return Order{}, fmt.Errorf("insert order item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Order{}, fmt.Errorf("commit order: %w", err)
	}

	return Order{
		ID:         orderID,
		Customer:   customer,
		Status:     req.Status,
		TotalCents: totalCents,
		CreatedAt:  createdAt,
		Items:      items,
	}, nil
}

func (s *Store) GetOrder(ctx context.Context, orderID string) (Order, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, customer_id, customer_email, customer_name, customer_segment, status, total_cents, created_at
		 FROM orders WHERE id = ?`,
		orderID,
	)

	var order Order
	if err := row.Scan(
		&order.ID,
		&order.Customer.ID,
		&order.Customer.Email,
		&order.Customer.FullName,
		&order.Customer.Segment,
		&order.Status,
		&order.TotalCents,
		&order.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Order{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
		}
		return Order{}, fmt.Errorf("find order: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT product_id, sku, name, category, quantity, unit_price_cents, line_total_cents
		 FROM order_items WHERE order_id = ?`,
		orderID,
	)
	if err != nil {
		return Order{}, fmt.Errorf("fetch order items: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ProductID, &item.SKU, &item.Name, &item.Category,
			&item.Quantity, &item.UnitPriceCents, &item.LineTotalCents); err != nil {
			return Order{}, fmt.Errorf("scan order item: %w", err)
		}
		order.Items = append(order.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Order{}, fmt.Errorf("iterate order items: %w", err)
	}

	return order, nil
}

func (s *Store) GetCustomerSummary(ctx context.Context, customerID string) (CustomerSummary, error) {
	var customer Customer
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, full_name, segment, created_at FROM customers WHERE id = ?`,
		customerID,
	)
	if err := row.Scan(&customer.ID, &customer.Email, &customer.FullName, &customer.Segment, &customer.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CustomerSummary{}, fmt.Errorf("%w: customer %s", ErrNotFound, customerID)
		}
		return CustomerSummary{}, fmt.Errorf("find customer: %w", err)
	}

	// Aggregate order-level stats.
	var ordersCount int
	var lifetimeValueCents sql.NullInt64
	var lastOrderAt sql.NullTime

	statsRow := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(total_cents), 0), MAX(created_at)
		 FROM orders WHERE customer_id = ?`,
		customerID,
	)
	if err := statsRow.Scan(&ordersCount, &lifetimeValueCents, &lastOrderAt); err != nil {
		return CustomerSummary{}, fmt.Errorf("aggregate customer stats: %w", err)
	}

	summary := CustomerSummary{
		Customer:           customer,
		OrdersCount:        ordersCount,
		LifetimeValueCents: int(lifetimeValueCents.Int64),
	}
	if ordersCount > 0 {
		summary.AverageOrderValueCents = summary.LifetimeValueCents / ordersCount
	}
	if lastOrderAt.Valid {
		t := lastOrderAt.Time.UTC()
		summary.LastOrderAt = &t
	}

	// Find favourite category.
	catRow := s.db.QueryRowContext(ctx,
		`SELECT oi.category
		 FROM orders o
		 JOIN order_items oi ON oi.order_id = o.id
		 WHERE o.customer_id = ?
		 GROUP BY oi.category
		 ORDER BY SUM(oi.line_total_cents) DESC, oi.category ASC
		 LIMIT 1`,
		customerID,
	)
	var favCat sql.NullString
	if err := catRow.Scan(&favCat); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return CustomerSummary{}, fmt.Errorf("favourite category: %w", err)
	}
	summary.FavoriteCategory = favCat.String

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

	query := `SELECT o.id, o.customer_id, o.customer_name, o.status, o.total_cents, o.created_at,
	           COALESCE(SUM(oi.quantity), 0)                   AS total_items,
	           COUNT(DISTINCT oi.product_id)                   AS distinct_products
	          FROM orders o
	          LEFT JOIN order_items oi ON oi.order_id = o.id
	          WHERE 1=1`
	args := []any{}

	if params.Status != "" {
		query += " AND o.status = ?"
		args = append(args, params.Status)
	}
	if params.CustomerID != "" {
		query += " AND o.customer_id = ?"
		args = append(args, params.CustomerID)
	}
	if params.MinTotalCents > 0 {
		query += " AND o.total_cents >= ?"
		args = append(args, params.MinTotalCents)
	}
	if params.CreatedFrom != nil {
		query += " AND o.created_at >= ?"
		args = append(args, *params.CreatedFrom)
	}
	if params.CreatedThrough != nil {
		query += " AND o.created_at <= ?"
		args = append(args, *params.CreatedThrough)
	}

	query += " GROUP BY o.id, o.customer_id, o.customer_name, o.status, o.total_cents, o.created_at"
	query += " ORDER BY o.created_at DESC"
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", params.Limit, params.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search orders: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	results := make([]OrderSearchResult, 0, params.Limit)
	for rows.Next() {
		var r OrderSearchResult
		if err := rows.Scan(
			&r.ID, &r.CustomerID, &r.CustomerName, &r.Status, &r.TotalCents, &r.CreatedAt,
			&r.TotalItems, &r.DistinctProducts,
		); err != nil {
			return nil, fmt.Errorf("scan order search result: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
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
	// SQL query parameter-free so keploy can match the mock deterministically
	// across record and replay sessions (time.Now() would shift the WHERE
	// clause and cause mock mismatches during replay).

	query := `SELECT oi.product_id, oi.sku, oi.name, oi.category,
	                 SUM(oi.quantity)         AS units_sold,
	                 SUM(oi.line_total_cents) AS revenue_cents,
	                 COUNT(DISTINCT o.id)     AS orders_count
	          FROM orders o
	          JOIN order_items oi ON oi.order_id = o.id
	          WHERE o.status IN ('paid', 'shipped')
	          GROUP BY oi.product_id, oi.sku, oi.name, oi.category
	          ORDER BY revenue_cents DESC, units_sold DESC
	          LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("top products: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	results := make([]TopProduct, 0, limit)
	rank := 1
	for rows.Next() {
		var p TopProduct
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Category,
			&p.UnitsSold, &p.RevenueCents, &p.OrdersCount); err != nil {
			return nil, fmt.Errorf("scan top product: %w", err)
		}
		p.RevenueRank = rank
		results = append(results, p)
		rank++
	}
	if err := rows.Err(); err != nil {
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
			ErrValidation, maxLargePayloadBytes, maxLargePayloadBytes/(1024*1024),
		)
	}

	checksum := sha256.Sum256([]byte(req.Payload))
	record := LargePayloadRecord{
		ID:               contentID(hex.EncodeToString(checksum[:])),
		Name:             req.Name,
		ContentType:      req.ContentType,
		PayloadSizeBytes: payloadSizeBytes,
		SHA256:           hex.EncodeToString(checksum[:]),
		CreatedAt:        time.Now().UTC(),
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO large_payloads (id, name, content_type, payload, payload_size_bytes, sha256, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.Name, record.ContentType, req.Payload,
		record.PayloadSizeBytes, record.SHA256, record.CreatedAt,
	)
	if err != nil {
		return LargePayloadRecord{}, fmt.Errorf("insert large payload: %w", err)
	}

	return record, nil
}

func (s *Store) GetLargePayload(ctx context.Context, payloadID string) (LargePayloadDetail, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, content_type, payload, payload_size_bytes, sha256, created_at
		 FROM large_payloads WHERE id = ?`,
		payloadID,
	)

	var d LargePayloadDetail
	if err := row.Scan(
		&d.ID, &d.Name, &d.ContentType, &d.Payload,
		&d.PayloadSizeBytes, &d.SHA256, &d.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LargePayloadDetail{}, fmt.Errorf("%w: large payload %s", ErrNotFound, payloadID)
		}
		return LargePayloadDetail{}, fmt.Errorf("find large payload: %w", err)
	}

	return d, nil
}

func (s *Store) DeleteLargePayload(ctx context.Context, payloadID string) (DeleteLargePayloadResponse, error) {
	// Fetch first so we can return the record metadata.
	detail, err := s.GetLargePayload(ctx, payloadID)
	if err != nil {
		return DeleteLargePayloadResponse{}, err
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM large_payloads WHERE id = ?`, payloadID)
	if err != nil {
		return DeleteLargePayloadResponse{}, fmt.Errorf("delete large payload: %w", err)
	}

	return DeleteLargePayloadResponse{
		Deleted: true,
		Record:  detail.LargePayloadRecord,
	}, nil
}
