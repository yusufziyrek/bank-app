#!/bin/bash

# --- Example Test Database Setup Script ---
# This script is for local/test use only. DO NOT use real passwords or secrets here.
# Edit the variables below with your own test credentials if needed.
#
# Usage:
#   bash test_db.example.sh
#
# Requirements: Docker must be installed and running.

CONTAINER_NAME="example-postgres-test"           # Name for the Docker container
POSTGRES_USER="exampleuser"                      # Dummy username (edit for your environment)
POSTGRES_PASSWORD="examplepass"                  # Dummy password (edit for your environment)
POSTGRES_DB="exampledb"                          # Dummy database name (edit for your environment)
HOST_PORT="6543"                                 # Host port to bind
CONTAINER_PORT="5432"                            # Default PostgreSQL port
IMAGE_NAME="postgres:17.5"                       # PostgreSQL Docker image

echo "Starting EXAMPLE PostgreSQL container setup..."

if [ "$(docker ps -aq -f name=$CONTAINER_NAME)" ]; then
  echo "Previous container detected. Stopping and removing..."
  docker rm -f $CONTAINER_NAME
fi

echo "Launching new PostgreSQL container..."
docker run --name $CONTAINER_NAME \
  -e POSTGRES_USER=$POSTGRES_USER \
  -e POSTGRES_PASSWORD=$POSTGRES_PASSWORD \
  -e POSTGRES_DB=$POSTGRES_DB \
  -p $HOST_PORT:$CONTAINER_PORT \
  -d $IMAGE_NAME

echo "Waiting for the database to start (5 seconds)..."
sleep 5

echo "Creating example database tables..."
docker exec -i $CONTAINER_NAME psql -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    full_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    account_number VARCHAR(30) UNIQUE NOT NULL,
    balance NUMERIC(12,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT REFERENCES accounts(id) ON DELETE CASCADE,
    to_account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    amount NUMERIC(12,2) NOT NULL,
    type VARCHAR(10) CHECK (type IN ('deposit', 'withdraw', 'transfer')) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cards (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT REFERENCES accounts(id) ON DELETE CASCADE,
    card_number VARCHAR(16) UNIQUE NOT NULL,
    expiry_date DATE NOT NULL,
    cvv VARCHAR(4) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id SERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  token TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL,
  CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
EOF

echo "✔ All example tables created successfully."
echo "EXAMPLE PostgreSQL container is now ready with the exampledb database."
