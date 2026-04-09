-- Seed data for Issue 3: Postgres large DataRow responses
--
-- The Keploy Postgres wire protocol parser fails when DataRow packets
-- exceed a single TCP segment (MSS ~1460 bytes on Docker bridge).
-- Error: "incomplete or invalid response packet (DataRow): want N bytes, have M"
--
-- Strategy: Make each individual row ~100KB so a SINGLE DataRow packet
-- spans dozens of TCP segments. Query 50 rows = 5MB+ response.
-- This guarantees TCP fragmentation even on localhost.

CREATE TABLE IF NOT EXISTS large_records (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    large_payload TEXT NOT NULL
);

-- Generate 200 rows, each with a ~100KB random payload.
-- A single DataRow at 100KB will span ~70 TCP segments (MSS=1460).
-- This forces the Postgres wire protocol parser to buffer across segments.
INSERT INTO large_records (name, description, large_payload)
SELECT
    'record-' || i,
    'Test record ' || i || ' — ' || repeat('description padding to increase row size ', 20),
    -- ~100KB per row: 64 chars per md5 pair × 1600 repeats = 102,400 chars
    repeat(
        md5(random()::text || i::text) || md5(random()::text || (i+1000)::text),
        1600
    )
FROM generate_series(1, 200) AS i;

-- Also create a table with many small columns (wide rows)
-- to trigger different DataRow encoding paths
CREATE TABLE IF NOT EXISTS wide_records (
    id SERIAL PRIMARY KEY,
    col_01 TEXT, col_02 TEXT, col_03 TEXT, col_04 TEXT, col_05 TEXT,
    col_06 TEXT, col_07 TEXT, col_08 TEXT, col_09 TEXT, col_10 TEXT,
    col_11 TEXT, col_12 TEXT, col_13 TEXT, col_14 TEXT, col_15 TEXT,
    col_16 TEXT, col_17 TEXT, col_18 TEXT, col_19 TEXT, col_20 TEXT
);

INSERT INTO wide_records (
    col_01, col_02, col_03, col_04, col_05,
    col_06, col_07, col_08, col_09, col_10,
    col_11, col_12, col_13, col_14, col_15,
    col_16, col_17, col_18, col_19, col_20
)
SELECT
    repeat(md5(random()::text), 100),  -- ~3.2KB per column × 20 = ~64KB per row
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100),
    repeat(md5(random()::text), 100)
FROM generate_series(1, 50) AS i;

-- Verify sizes
SELECT 'large_records' AS tbl,
    count(*) AS rows,
    pg_size_pretty(avg(length(large_payload)::bigint)) AS avg_payload,
    pg_size_pretty(sum(length(large_payload)::bigint)) AS total_payload
FROM large_records
UNION ALL
SELECT 'wide_records',
    count(*),
    pg_size_pretty(avg(octet_length(col_01)::bigint)),
    pg_size_pretty(sum(octet_length(col_01)::bigint) * 20)
FROM wide_records;
