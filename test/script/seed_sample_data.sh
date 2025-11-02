#!/bin/bash

# Basit test verisi üretir. Çalıştırmadan önce test_db.sh ile PostgreSQL konteynerini başlatın.
# Kart tablosu şifrelenmiş PAN beklediğinden bu script kart eklemez; kartları API üzerinden oluşturun.

set -euo pipefail

CONTAINER_NAME="${CONTAINER_NAME:-postgres-test}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-bankapp}"

DEFAULT_PASSWORD_HASH="\$2a\$10\$FsePQ4j/wcgUp3NEPoPFcOJ4i/hTyBC0b.poKT1eZq9VHtVxo1cW."
ADMIN_PASSWORD_HASH="\$2a\$10\$sqjEfak/I.QxkFhuwskjtueLgbF5XGodJowEeB23Tep75PfmPXTt6"

if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  echo "Hata: ${CONTAINER_NAME} konteyneri çalışmıyor. Önce test_db.sh scriptini çalıştırın." >&2
  exit 1
fi

RESET_SCHEMA="${RESET_SCHEMA:-true}"

SQL=""

if [[ "${RESET_SCHEMA}" == "true" ]]; then
  SQL+="TRUNCATE TABLE transactions, cards, accounts, users, refresh_tokens RESTART IDENTITY CASCADE;\n"
fi

SQL+="\n"

SQL+="WITH base_users AS (\n"
SQL+="    INSERT INTO users (full_name, email, password_hash, role, is_active, created_at, updated_at) VALUES\n"
SQL+="      ('Demo Kullanıcı 1', 'demo1@example.com', '${DEFAULT_PASSWORD_HASH}', 'user', TRUE, NOW() - INTERVAL '40 days', NOW() - INTERVAL '1 day'),\n"
SQL+="      ('Demo Kullanıcı 2', 'demo2@example.com', '${DEFAULT_PASSWORD_HASH}', 'user', TRUE, NOW() - INTERVAL '25 days', NOW() - INTERVAL '12 hours'),\n"
SQL+="      ('Demo Admin', 'admin@example.com', '${ADMIN_PASSWORD_HASH}', 'admin', TRUE, NOW() - INTERVAL '60 days', NOW())\n"
SQL+="    ON CONFLICT (email) DO UPDATE SET full_name = EXCLUDED.full_name\n"
SQL+="    RETURNING id, email, created_at\n"
SQL+="), random_users AS (\n"
SQL+="    SELECT g AS seq,\n"
SQL+="           concat('Load Test User ', g) AS full_name,\n"
SQL+="           concat('loadtest', lpad(g::text, 3, '0'), '@example.com') AS email,\n"
SQL+="           '${DEFAULT_PASSWORD_HASH}' AS password_hash,\n"
SQL+="           NOW() - (g || ' days')::interval AS created_at,\n"
SQL+="           NOW() - ((g % 24) || ' hours')::interval AS updated_at\n"
SQL+="    FROM generate_series(1, 75) g\n"
SQL+="), bulk_users AS (\n"
SQL+="    INSERT INTO users (full_name, email, password_hash, role, is_active, created_at, updated_at)\n"
SQL+="    SELECT full_name, email, password_hash, 'user', TRUE, created_at, updated_at\n"
SQL+="    FROM random_users\n"
SQL+="    ON CONFLICT (email) DO UPDATE SET updated_at = EXCLUDED.updated_at\n"
SQL+="    RETURNING id, email, created_at\n"
SQL+="), all_users AS (\n"
SQL+="    SELECT id, email, created_at FROM base_users\n"
SQL+="    UNION ALL\n"
SQL+="    SELECT id, email, created_at FROM bulk_users\n"
SQL+="), account_source AS (\n"
SQL+="    SELECT u.id AS user_id,\n"
SQL+="           row_number() OVER (ORDER BY u.id, acc_idx) AS account_seq,\n"
SQL+="           acc_idx\n"
SQL+="    FROM all_users u\n"
SQL+="    CROSS JOIN generate_series(1, 3) AS acc_idx\n"
SQL+="), inserted_accounts AS (\n"
SQL+="    INSERT INTO accounts (user_id, account_number, balance, created_at, updated_at)\n"
SQL+="    SELECT user_id,\n"
SQL+="           concat('TR', lpad((100000000000000000 + account_seq)::text, 18, '0')) AS account_number,\n"
SQL+="           ROUND((random() * 9000 + 100)::numeric, 2) AS balance,\n"
SQL+="           NOW() - ((account_seq % 45) || ' days')::interval AS created_at,\n"
SQL+="           NOW() - ((account_seq % 72) || ' hours')::interval AS updated_at\n"
SQL+="    FROM account_source\n"
SQL+="    RETURNING id, user_id, account_number, created_at, updated_at\n"
SQL+="), account_list AS (\n"
SQL+="    SELECT id, user_id, account_number, created_at, updated_at,\n"
SQL+="           LEAD(id) OVER (ORDER BY id) AS next_account_id\n"
SQL+="    FROM inserted_accounts\n"
SQL+="), transaction_source AS (\n"
SQL+="    SELECT a.id AS account_id,\n"
SQL+="           CASE WHEN g.i % 3 = 0 THEN COALESCE(a.next_account_id, a.id) ELSE NULL END AS to_account_id,\n"
SQL+="           ROUND((random() * 450 + 50)::numeric, 2) AS amount,\n"
SQL+="           CASE WHEN g.i % 3 = 0 THEN 'transfer' WHEN g.i % 3 = 1 THEN 'deposit' ELSE 'withdraw' END AS type,\n"
SQL+="           CASE WHEN g.i % 3 = 0 THEN 'Load test transfer' WHEN g.i % 3 = 1 THEN 'Load test deposit' ELSE 'Load test withdraw' END AS description,\n"
SQL+="           a.created_at + (g.i || ' days')::interval AS created_at\n"
SQL+="    FROM account_list a\n"
SQL+="    CROSS JOIN generate_series(1, 12) AS g(i)\n"
SQL+=")\n"
SQL+="INSERT INTO transactions (account_id, to_account_id, amount, type, description, created_at)\n"
SQL+="SELECT account_id,\n"
SQL+="       CASE WHEN to_account_id = account_id THEN NULL ELSE to_account_id END,\n"
SQL+="       amount, type, description, created_at\n"
SQL+="FROM transaction_source;\n"
SQL+="\n"

SQL+="INSERT INTO refresh_tokens (user_id, token, expires_at, created_at)\n"
SQL+="SELECT id, concat('seed-refresh-token-', id), NOW() + INTERVAL '14 days', NOW()\n"
SQL+="FROM users\n"
SQL+="WHERE email IN ('demo1@example.com', 'demo2@example.com', 'admin@example.com')\n"
SQL+="ON CONFLICT (token) DO NOTHING;\n"

# psql içindeki \x başka bir katmanda yorumlanmasın diye printf kullanıyoruz.
printf -v SQL "%b" "$SQL"

docker exec -i "$CONTAINER_NAME" psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" <<EOF
$SQL
EOF

echo "✔ Test verileri başarıyla eklendi"
