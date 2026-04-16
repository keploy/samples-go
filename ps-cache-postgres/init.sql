CREATE SCHEMA IF NOT EXISTS travelcard;

CREATE TABLE IF NOT EXISTS travelcard.travel_account (
    id        SERIAL PRIMARY KEY,
    member_id INT    NOT NULL UNIQUE,
    name      TEXT   NOT NULL,
    balance   INT    NOT NULL DEFAULT 0
);

INSERT INTO travelcard.travel_account (member_id, name, balance) VALUES
    (19, 'Alice',   1000),
    (23, 'Bob',     2500),
    (31, 'Charlie', 500),
    (42, 'Diana',   7500)
ON CONFLICT (member_id) DO NOTHING;
