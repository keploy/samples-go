CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS customers (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    full_name TEXT NOT NULL,
    segment TEXT NOT NULL CHECK (segment IN ('startup', 'enterprise', 'retail', 'partner')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    sku TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price_cents INTEGER NOT NULL CHECK (price_cents > 0),
    inventory_count INTEGER NOT NULL DEFAULT 0 CHECK (inventory_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    status TEXT NOT NULL CHECK (status IN ('pending', 'paid', 'shipped', 'cancelled')),
    total_cents INTEGER NOT NULL DEFAULT 0 CHECK (total_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products (id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_cents INTEGER NOT NULL CHECK (unit_price_cents > 0),
    line_total_cents INTEGER GENERATED ALWAYS AS (quantity * unit_price_cents) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_customer_created_at
    ON orders (customer_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_orders_status_created_at
    ON orders (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id
    ON order_items (order_id);

CREATE INDEX IF NOT EXISTS idx_order_items_product_id
    ON order_items (product_id);

CREATE INDEX IF NOT EXISTS idx_products_category
    ON products (category);

CREATE TABLE IF NOT EXISTS large_payloads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    payload_text TEXT NOT NULL,
    payload_size_bytes INTEGER NOT NULL CHECK (payload_size_bytes > 0),
    sha256 TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_large_payloads_created_at
    ON large_payloads (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_large_payloads_payload_size_bytes
    ON large_payloads (payload_size_bytes DESC);
