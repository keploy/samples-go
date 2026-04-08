package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
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

	const query = `
		INSERT INTO customers (email, full_name, segment)
		VALUES ($1, $2, $3)
		RETURNING id, email, full_name, segment, created_at;
	`

	var customer Customer
	err := s.db.QueryRowContext(ctx, query, req.Email, req.FullName, req.Segment).Scan(
		&customer.ID,
		&customer.Email,
		&customer.FullName,
		&customer.Segment,
		&customer.CreatedAt,
	)
	if err != nil {
		if isPgCode(err, "23505") {
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

	const query = `
		INSERT INTO products (sku, name, category, price_cents, inventory_count)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, sku, name, category, price_cents, inventory_count, created_at;
	`

	var product Product
	err := s.db.QueryRowContext(
		ctx,
		query,
		req.SKU,
		req.Name,
		req.Category,
		req.PriceCents,
		req.InventoryCount,
	).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Category,
		&product.PriceCents,
		&product.InventoryCount,
		&product.CreatedAt,
	)
	if err != nil {
		if isPgCode(err, "23505") {
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
	case req.CustomerID <= 0:
		return Order{}, fmt.Errorf("%w: customer_id must be greater than zero", ErrValidation)
	case len(req.Items) == 0:
		return Order{}, fmt.Errorf("%w: at least one item is required", ErrValidation)
	}

	if _, ok := validStatuses[req.Status]; !ok {
		return Order{}, fmt.Errorf("%w: unsupported order status", ErrValidation)
	}

	for _, item := range req.Items {
		if item.ProductID <= 0 || item.Quantity <= 0 {
			return Order{}, fmt.Errorf("%w: every item needs a valid product_id and quantity", ErrValidation)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var customerExists bool
	if err := tx.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM customers WHERE id = $1)`,
		req.CustomerID,
	).Scan(&customerExists); err != nil {
		return Order{}, fmt.Errorf("check customer: %w", err)
	}
	if !customerExists {
		return Order{}, fmt.Errorf("%w: customer %d", ErrNotFound, req.CustomerID)
	}

	var orderID string
	var createdAt time.Time
	if err := tx.QueryRowContext(
		ctx,
		`INSERT INTO orders (customer_id, status) VALUES ($1, $2) RETURNING id::text, created_at`,
		req.CustomerID,
		req.Status,
	).Scan(&orderID, &createdAt); err != nil {
		return Order{}, fmt.Errorf("insert order: %w", err)
	}

	totalCents := 0
	for _, item := range req.Items {
		var priceCents int
		var inventoryCount int
		if err := tx.QueryRowContext(
			ctx,
			`SELECT price_cents, inventory_count FROM products WHERE id = $1 FOR UPDATE`,
			item.ProductID,
		).Scan(&priceCents, &inventoryCount); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Order{}, fmt.Errorf("%w: product %d", ErrNotFound, item.ProductID)
			}
			return Order{}, fmt.Errorf("select product %d: %w", item.ProductID, err)
		}

		if inventoryCount < item.Quantity {
			return Order{}, fmt.Errorf("%w: product %d only has %d units available", ErrInsufficientInventory, item.ProductID, inventoryCount)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents) VALUES ($1::uuid, $2, $3, $4)`,
			orderID,
			item.ProductID,
			item.Quantity,
			priceCents,
		); err != nil {
			return Order{}, fmt.Errorf("insert order item: %w", err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`UPDATE products SET inventory_count = inventory_count - $1 WHERE id = $2`,
			item.Quantity,
			item.ProductID,
		); err != nil {
			return Order{}, fmt.Errorf("update inventory: %w", err)
		}

		totalCents += item.Quantity * priceCents
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE orders SET total_cents = $1 WHERE id = $2::uuid`,
		totalCents,
		orderID,
	); err != nil {
		return Order{}, fmt.Errorf("update order total: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Order{}, fmt.Errorf("commit order transaction: %w", err)
	}

	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return Order{}, fmt.Errorf("reload order after insert: %w", err)
	}

	order.CreatedAt = createdAt
	return order, nil
}

func (s *Store) GetOrder(ctx context.Context, orderID string) (Order, error) {
	const query = `
		SELECT
			o.id::text,
			o.status,
			o.total_cents,
			o.created_at,
			c.id,
			c.email,
			c.full_name,
			c.segment,
			c.created_at,
			COALESCE(oi.product_id, 0),
			COALESCE(p.sku, ''),
			COALESCE(p.name, ''),
			COALESCE(p.category, ''),
			COALESCE(oi.quantity, 0),
			COALESCE(oi.unit_price_cents, 0),
			COALESCE(oi.line_total_cents, 0)
		FROM orders o
		JOIN customers c ON c.id = o.customer_id
		LEFT JOIN order_items oi ON oi.order_id = o.id
		LEFT JOIN products p ON p.id = oi.product_id
		WHERE o.id = $1::uuid
		ORDER BY oi.id;
	`

	rows, err := s.db.QueryContext(ctx, query, orderID)
	if err != nil {
		if isPgCode(err, "22P02") {
			return Order{}, fmt.Errorf("%w: invalid order id", ErrValidation)
		}
		return Order{}, fmt.Errorf("query order: %w", err)
	}
	defer rows.Close()

	var order Order
	found := false

	for rows.Next() {
		found = true

		var item OrderItem
		if err := rows.Scan(
			&order.ID,
			&order.Status,
			&order.TotalCents,
			&order.CreatedAt,
			&order.Customer.ID,
			&order.Customer.Email,
			&order.Customer.FullName,
			&order.Customer.Segment,
			&order.Customer.CreatedAt,
			&item.ProductID,
			&item.SKU,
			&item.Name,
			&item.Category,
			&item.Quantity,
			&item.UnitPriceCents,
			&item.LineTotalCents,
		); err != nil {
			return Order{}, fmt.Errorf("scan order row: %w", err)
		}

		if item.ProductID != 0 {
			order.Items = append(order.Items, item)
		}
	}

	if err := rows.Err(); err != nil {
		return Order{}, fmt.Errorf("iterate order rows: %w", err)
	}
	if !found {
		return Order{}, fmt.Errorf("%w: order %s", ErrNotFound, orderID)
	}

	return order, nil
}

func (s *Store) GetCustomerSummary(ctx context.Context, customerID int64) (CustomerSummary, error) {
	const query = `
		WITH customer_orders AS (
			SELECT o.id, o.customer_id, o.total_cents, o.created_at
			FROM orders o
			WHERE o.customer_id = $1
		),
		category_spend AS (
			SELECT p.category, SUM(oi.line_total_cents) AS spend_cents
			FROM customer_orders co
			JOIN order_items oi ON oi.order_id = co.id
			JOIN products p ON p.id = oi.product_id
			GROUP BY p.category
		),
		ranked_categories AS (
			SELECT
				category,
				spend_cents,
				ROW_NUMBER() OVER (ORDER BY spend_cents DESC, category ASC) AS category_rank
			FROM category_spend
		)
		SELECT
			c.id,
			c.email,
			c.full_name,
			c.segment,
			c.created_at,
			COUNT(co.id) AS orders_count,
			COALESCE(SUM(co.total_cents), 0) AS lifetime_value_cents,
			COALESCE(ROUND(AVG(co.total_cents)), 0)::bigint AS average_order_value_cents,
			MAX(co.created_at) AS last_order_at,
			rc.category
		FROM customers c
		LEFT JOIN customer_orders co ON co.customer_id = c.id
		LEFT JOIN ranked_categories rc ON rc.category_rank = 1
		WHERE c.id = $1
		GROUP BY c.id, rc.category;
	`

	var summary CustomerSummary
	var lastOrderAt sql.NullTime
	var favoriteCategory sql.NullString

	err := s.db.QueryRowContext(ctx, query, customerID).Scan(
		&summary.Customer.ID,
		&summary.Customer.Email,
		&summary.Customer.FullName,
		&summary.Customer.Segment,
		&summary.Customer.CreatedAt,
		&summary.OrdersCount,
		&summary.LifetimeValueCents,
		&summary.AverageOrderValueCents,
		&lastOrderAt,
		&favoriteCategory,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CustomerSummary{}, fmt.Errorf("%w: customer %d", ErrNotFound, customerID)
		}
		return CustomerSummary{}, fmt.Errorf("query customer summary: %w", err)
	}

	if lastOrderAt.Valid {
		summary.LastOrderAt = &lastOrderAt.Time
	}
	if favoriteCategory.Valid {
		summary.FavoriteCategory = favoriteCategory.String
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

	var fromArg any
	if params.CreatedFrom != nil {
		fromArg = *params.CreatedFrom
	}
	var throughArg any
	if params.CreatedThrough != nil {
		throughArg = *params.CreatedThrough
	}

	const query = `
		WITH filtered_orders AS (
			SELECT o.id, o.customer_id, o.status, o.total_cents, o.created_at
			FROM orders o
			WHERE ($1::text = '' OR o.status = $1)
			  AND ($2::bigint = 0 OR o.customer_id = $2)
			  AND ($3::integer = 0 OR o.total_cents >= $3)
			  AND ($4::timestamptz IS NULL OR o.created_at >= $4)
			  AND ($5::timestamptz IS NULL OR o.created_at <= $5)
			ORDER BY o.created_at DESC
			LIMIT $6 OFFSET $7
		),
		item_rollup AS (
			SELECT
				oi.order_id,
				SUM(oi.quantity) AS total_items,
				COUNT(*) AS distinct_products
			FROM order_items oi
			WHERE oi.order_id IN (SELECT id FROM filtered_orders)
			GROUP BY oi.order_id
		)
		SELECT
			fo.id::text,
			fo.customer_id,
			c.full_name,
			fo.status,
			fo.total_cents,
			fo.created_at,
			COALESCE(ir.total_items, 0),
			COALESCE(ir.distinct_products, 0)
		FROM filtered_orders fo
		JOIN customers c ON c.id = fo.customer_id
		LEFT JOIN item_rollup ir ON ir.order_id = fo.id
		ORDER BY fo.created_at DESC;
	`

	rows, err := s.db.QueryContext(
		ctx,
		query,
		params.Status,
		params.CustomerID,
		params.MinTotalCents,
		fromArg,
		throughArg,
		params.Limit,
		params.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("query order search: %w", err)
	}
	defer rows.Close()

	results := make([]OrderSearchResult, 0, params.Limit)
	for rows.Next() {
		var row OrderSearchResult
		if err := rows.Scan(
			&row.ID,
			&row.CustomerID,
			&row.CustomerName,
			&row.Status,
			&row.TotalCents,
			&row.CreatedAt,
			&row.TotalItems,
			&row.DistinctProducts,
		); err != nil {
			return nil, fmt.Errorf("scan order search row: %w", err)
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order search rows: %w", err)
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

	const query = `
		WITH recent_orders AS (
			SELECT id
			FROM orders
			WHERE status IN ('paid', 'shipped')
			  AND created_at >= NOW() - ($1 * INTERVAL '1 day')
		),
		product_totals AS (
			SELECT
				p.id,
				p.sku,
				p.name,
				p.category,
				SUM(oi.quantity) AS units_sold,
				SUM(oi.line_total_cents) AS revenue_cents,
				COUNT(DISTINCT oi.order_id) AS orders_count
			FROM recent_orders ro
			JOIN order_items oi ON oi.order_id = ro.id
			JOIN products p ON p.id = oi.product_id
			GROUP BY p.id, p.sku, p.name, p.category
		),
		ranked_products AS (
			SELECT
				id,
				sku,
				name,
				category,
				units_sold,
				revenue_cents,
				orders_count,
				DENSE_RANK() OVER (
					ORDER BY revenue_cents DESC, units_sold DESC, id ASC
				) AS revenue_rank
			FROM product_totals
		)
		SELECT
			id,
			sku,
			name,
			category,
			units_sold,
			revenue_cents,
			orders_count,
			revenue_rank
		FROM ranked_products
		ORDER BY revenue_rank ASC, id ASC
		LIMIT $2;
	`

	rows, err := s.db.QueryContext(ctx, query, days, limit)
	if err != nil {
		return nil, fmt.Errorf("query top products: %w", err)
	}
	defer rows.Close()

	results := make([]TopProduct, 0, limit)
	for rows.Next() {
		var row TopProduct
		if err := rows.Scan(
			&row.ID,
			&row.SKU,
			&row.Name,
			&row.Category,
			&row.UnitsSold,
			&row.RevenueCents,
			&row.OrdersCount,
			&row.RevenueRank,
		); err != nil {
			return nil, fmt.Errorf("scan top product row: %w", err)
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top products rows: %w", err)
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

	const query = `
		INSERT INTO large_payloads (
			name,
			content_type,
			payload_text,
			payload_size_bytes,
			sha256
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, name, content_type, payload_size_bytes, sha256, created_at;
	`

	var record LargePayloadRecord
	err := s.db.QueryRowContext(
		ctx,
		query,
		req.Name,
		req.ContentType,
		req.Payload,
		payloadSizeBytes,
		hex.EncodeToString(checksum[:]),
	).Scan(
		&record.ID,
		&record.Name,
		&record.ContentType,
		&record.PayloadSizeBytes,
		&record.SHA256,
		&record.CreatedAt,
	)
	if err != nil {
		return LargePayloadRecord{}, fmt.Errorf("insert large payload: %w", err)
	}

	return record, nil
}

func (s *Store) GetLargePayload(ctx context.Context, payloadID string) (LargePayloadDetail, error) {
	const query = `
		SELECT
			id::text,
			name,
			content_type,
			payload_text,
			payload_size_bytes,
			sha256,
			created_at
		FROM large_payloads
		WHERE id = $1::uuid;
	`

	var record LargePayloadDetail
	err := s.db.QueryRowContext(ctx, query, payloadID).Scan(
		&record.ID,
		&record.Name,
		&record.ContentType,
		&record.Payload,
		&record.PayloadSizeBytes,
		&record.SHA256,
		&record.CreatedAt,
	)
	if err != nil {
		if isPgCode(err, "22P02") {
			return LargePayloadDetail{}, fmt.Errorf("%w: invalid payload id", ErrValidation)
		}
		if errors.Is(err, sql.ErrNoRows) {
			return LargePayloadDetail{}, fmt.Errorf("%w: large payload %s", ErrNotFound, payloadID)
		}
		return LargePayloadDetail{}, fmt.Errorf("query large payload: %w", err)
	}

	return record, nil
}

func (s *Store) DeleteLargePayload(ctx context.Context, payloadID string) (DeleteLargePayloadResponse, error) {
	const query = `
		DELETE FROM large_payloads
		WHERE id = $1::uuid
		RETURNING id::text, name, content_type, payload_size_bytes, sha256, created_at;
	`

	var record LargePayloadRecord
	err := s.db.QueryRowContext(ctx, query, payloadID).Scan(
		&record.ID,
		&record.Name,
		&record.ContentType,
		&record.PayloadSizeBytes,
		&record.SHA256,
		&record.CreatedAt,
	)
	if err != nil {
		if isPgCode(err, "22P02") {
			return DeleteLargePayloadResponse{}, fmt.Errorf("%w: invalid payload id", ErrValidation)
		}
		if errors.Is(err, sql.ErrNoRows) {
			return DeleteLargePayloadResponse{}, fmt.Errorf("%w: large payload %s", ErrNotFound, payloadID)
		}
		return DeleteLargePayloadResponse{}, fmt.Errorf("delete large payload: %w", err)
	}

	return DeleteLargePayloadResponse{
		Deleted: true,
		Record:  record,
	}, nil
}

func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
