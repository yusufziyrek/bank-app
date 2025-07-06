#!/bin/bash

# Config
CONTAINER_NAME="postgres-test"
POSTGRES_USER="postgres"
POSTGRES_PASSWORD="1234"
POSTGRES_DB="bankapp"
HOST_PORT="6432"
CONTAINER_PORT="5432"
IMAGE_NAME="postgres:17.5"

# Önceki konteyner varsa sil
if [ "$(docker ps -aq -f name=$CONTAINER_NAME)" ]; then
  echo "Önceki konteyner tespit edildi. Durduruluyor ve siliniyor..."
  docker rm -f $CONTAINER_NAME
fi

# PostgreSQL konteyneri başlat
echo "Yeni PostgreSQL konteyneri başlatılıyor..."
docker run --name $CONTAINER_NAME \
  -e POSTGRES_USER=$POSTGRES_USER \
  -e POSTGRES_PASSWORD=$POSTGRES_PASSWORD \
  -e POSTGRES_DB=$POSTGRES_DB \
  -p $HOST_PORT:$CONTAINER_PORT \
  -d $IMAGE_NAME

# Veritabanı hazır olana kadar bekle
echo "Veritabanı başlatılıyor, lütfen bekleyin..."
sleep 5

# Veritabanı oluştur (güvenlik için)
docker exec -i $CONTAINER_NAME psql -U $POSTGRES_USER -d postgres -c "CREATE DATABASE $POSTGRES_DB;" 2>/dev/null || echo "Veritabanı zaten mevcut."

# Tabloları oluştur
docker exec -i $CONTAINER_NAME psql -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    full_name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    account_number VARCHAR(20) UNIQUE NOT NULL,
    balance NUMERIC(12,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id) ON DELETE CASCADE,
    amount NUMERIC(12,2) NOT NULL,
    type VARCHAR(10) CHECK (type IN ('deposit', 'withdraw')) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cards (
    id SERIAL PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id) ON DELETE CASCADE,
    card_number VARCHAR(16) UNIQUE NOT NULL,
    expiry_date DATE NOT NULL,
    cvv VARCHAR(4) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
EOF

echo "✔ Tüm tablolar başarıyla oluşturuldu ✅"
