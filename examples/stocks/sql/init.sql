CREATE DATABASE IF NOT EXISTS stocks;
USE stocks;

CREATE TABLE IF NOT EXISTS stocks (
    date Date MATERIALIZED timestamp,
    timestamp DateTime64(9),

    figi String,
    ticker String,
    currency String,
    price_rub Float64,
    price_usd Float64
) ENGINE = ReplacingMergeTree()
    PARTITION BY date
    ORDER BY (timestamp, figi)
    TTL date + INTERVAL 30 DAY DELETE;
