#!/usr/bin/env bash
set -euo pipefail

KAFKA_CONTAINER="${KAFKA_CONTAINER:-order-service-kafka-1}"
BROKER="${BROKER:-localhost:9092}"
TOPIC="${TOPIC:-orders}"

produce() {
  local key="$1"
  local compact
  compact="$(tr -d '\n' < /dev/stdin)"
  printf '%s\n' "${key}:${compact}" \
  | docker exec -i "$KAFKA_CONTAINER" kafka-console-producer \
      --broker-list "$BROKER" \
      --topic "$TOPIC" \
      --property parse.key=true \
      --property key.separator=:
  echo "sent key=${key}"
}

echo "Publishing 4 messages to Kafka (topic=$TOPIC, broker=$BROKER)..."

# 1) Валидный — как в задании
produce "b563feb7b2b84b6test" <<'JSON'
{
  "order_uid": "b563feb7b2b84b6test",
  "track_number": "WBILMTESTTRACK",
  "entry": "WBIL",
  "delivery": {
    "name": "Test Testov",
    "phone": "+9720000000",
    "zip": "2639809",
    "city": "Kiryat Mozkin",
    "address": "Ploshad Mira 15",
    "region": "Kraiot",
    "email": "test@gmail.com"
  },
  "payment": {
    "transaction": "b563feb7b2b84b6test",
    "request_id": "",
    "currency": "USD",
    "provider": "wbpay",
    "amount": 1817,
    "payment_dt": 1637907727,
    "bank": "alpha",
    "delivery_cost": 1500,
    "goods_total": 317,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": 9934930,
      "track_number": "WBILMTESTTRACK",
      "price": 453,
      "rid": "ab4219087a764ae0btest",
      "name": "Mascaras",
      "sale": 30,
      "size": "0",
      "total_price": 317,
      "nm_id": 2389212,
      "brand": "Vivienne Sabo",
      "status": 202
    }
  ],
  "locale": "en",
  "internal_signature": "",
  "customer_id": "test",
  "delivery_service": "meest",
  "shardkey": "9",
  "sm_id": 99,
  "date_created": "2021-11-26T06:22:19Z",
  "oof_shard": "1"
}
JSON

# 2) Валидный — с двумя товарами
produce "ok_multi_items_1" <<'JSON'
{
  "order_uid": "ok_multi_items_1",
  "track_number": "WBILMTESTTRACK2",
  "entry": "WBIL",
  "delivery": {
    "name": "Ivan Ivanov",
    "phone": "+79001234567",
    "zip": "123456",
    "city": "Moscow",
    "address": "Red Square 1",
    "region": "Moscow",
    "email": "ivan@example.com"
  },
  "payment": {
    "transaction": "ok_multi_items_1",
    "request_id": "req123",
    "currency": "RUB",
    "provider": "wbpay",
    "amount": 5000,
    "payment_dt": 1637907727,
    "bank": "sberbank",
    "delivery_cost": 300,
    "goods_total": 4700,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": 1111111,
      "track_number": "WBILMTESTTRACK2",
      "price": 2000,
      "rid": "item1",
      "name": "T-Shirt",
      "sale": 10,
      "size": "L",
      "total_price": 1800,
      "nm_id": 111111,
      "brand": "Nike",
      "status": 202
    },
    {
      "chrt_id": 2222222,
      "track_number": "WBILMTESTTRACK2",
      "price": 3000,
      "rid": "item2",
      "name": "Jeans",
      "sale": 5,
      "size": "32",
      "total_price": 2850,
      "nm_id": 222222,
      "brand": "Levis",
      "status": 202
    }
  ],
  "locale": "ru",
  "internal_signature": "",
  "customer_id": "customer123",
  "delivery_service": "cdek",
  "shardkey": "5",
  "sm_id": 88,
  "date_created": "2023-12-01T10:00:00Z",
  "oof_shard": "2"
}
JSON

# 3) Невалидный — transaction != order_uid
produce "mismatch_tx" <<'JSON'
{
  "order_uid": "mismatch_tx",
  "track_number": "TRK_MIS",
  "entry": "WBIL",
  "delivery": {
    "name": "M",
    "phone": "+0",
    "zip": "0",
    "city": "X",
    "address": "Y",
    "region": "Z",
    "email": "a@b.c"
  },
  "payment": {
    "transaction": "other_id",
    "request_id": "",
    "currency": "USD",
    "provider": "wbpay",
    "amount": 10,
    "payment_dt": 1637907727,
    "bank": "b",
    "delivery_cost": 1,
    "goods_total": 9,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": 1,
      "track_number": "TRK_MIS",
      "price": 9,
      "rid": "r",
      "name": "n",
      "sale": 0,
      "size": "s",
      "total_price": 9,
      "nm_id": 1,
      "brand": "b",
      "status": 1
    }
  ],
  "locale": "en",
  "internal_signature": "",
  "customer_id": "c",
  "delivery_service": "svc",
  "shardkey": "1",
  "sm_id": 1,
  "date_created": "2023-12-02T10:00:00Z",
  "oof_shard": "1"
}
JSON

# 4) Невалидный — сломанный JSON
produce "bad_json" <<'JSON'
{
  "order_uid": "bad_json"
  "track_number": "X"
}
JSON

echo "Done."