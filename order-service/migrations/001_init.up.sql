CREATE TABLE IF NOT EXISTS orders (
    order_uid VARCHAR(255) PRIMARY KEY,
    track_number VARCHAR(255) NOT NULL,
    entry VARCHAR(50) NOT NULL,
    locale VARCHAR(10),
    internal_signature TEXT,
    customer_id VARCHAR(255),
    delivery_service VARCHAR(255) NOT NULL,
    shardkey VARCHAR(10),
    sm_id INTEGER,
    date_created TIMESTAMPTZ DEFAULT NOW(),
    oof_shard VARCHAR(10)
);

CREATE TABLE IF NOT EXISTS delivery (
    order_id VARCHAR(255) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(255) NOT NULL,
    zip VARCHAR(255) NOT NULL,
    city VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    region VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS payments (
    order_id VARCHAR(255) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    transaction VARCHAR(255) NOT NULL,
    request_id VARCHAR(255),
    currency VARCHAR(10) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    amount INTEGER NOT NULL CHECK (amount >= 0),
    payment_dt BIGINT NOT NULL,
    bank VARCHAR(50),
    delivery_cost INTEGER NOT NULL CHECK (delivery_cost >= 0),
    goods_total INTEGER NOT NULL CHECK (goods_total >= 0),
    custom_fee INTEGER NOT NULL CHECK (custom_fee >= 0)
);

CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    order_id VARCHAR(255) NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    chrt_id BIGINT NOT NULL,
    track_number VARCHAR(255),
    price INTEGER NOT NULL CHECK (price >= 0),
    rid VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    sale INTEGER NOT NULL CHECK (sale >= 0),
    size VARCHAR(50),
    total_price INTEGER NOT NULL CHECK (total_price >= 0),
    nm_id BIGINT NOT NULL,
    brand VARCHAR(255),
    status INTEGER
);