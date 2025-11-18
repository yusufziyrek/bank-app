### BankApp-RestAPI Documentation

**Overview:**
BankApp-RestAPI is a secure, high-performance backend service written in Go that provides essential banking operations. It offers comprehensive CRUD operations for users, accounts, cards, and transactions—each protected via JWT authentication. Card data is protected with AES-GCM (PAN encrypted at rest, CVV hashed) and masked in responses. Dockerized PostgreSQL support and automated setup scripts make development and testing seamless. Optional Redis caching and in-memory rate limiting are available.

---

### **1. Features:**

* **User Management:**

  * Register, update, delete users
  * List and retrieve user profiles
* **Account Management:**

  * Create, read, update, delete bank accounts
  * Link multiple accounts per user
* **Card Management:**

  * Issue, renew, deactivate, and revoke cards
  * Secure storage and masked responses
* **Transaction Processing:**

  * Deposit, withdraw, and transfer funds
  * Retrieve transaction history
* **JWT Authentication & Authorization:**

  * Token-based security with role distinctions (admin vs. user)
* **Dockerized Testing Environment:**

  * Spin up PostgreSQL via script
  * Automated schema initialization
* **Caching Layer:**

  * Optional Redis-backed user profile cache

---

### **2. Tech Stack:**

* **Go 1.25+**
* **Echo Framework** for HTTP routing and middleware
* **JWT** (`github.com/golang-jwt/jwt`) for token handling
* **PostgreSQL** as the relational database
* **Redis** for caching frequently fetched data
* **Docker & Shell Scripts** for containerized setup

---

### **3. Architecture:**

* **Layered Design:**

  * **Controller Layer:** HTTP handlers & request validation
  * **Service Layer:** Business logic & transaction orchestration
  * **Repository Layer:** Database queries & persistence
  * **Model/DTO Layer:** Domain models & transfer objects
* **Modular Packaging:** Separate packages for `auth`, `users`, `accounts`, `cards`, and `transactions`.

---
### **4. Database Structure**

* **`users`**
  `id`, `full_name`, `email`, `password_hash`, `role`, `is_active`, `created_at`, `updated_at`

* **`accounts`**
  `id`, `user_id`, `account_number`, `balance`, `created_at`, `updated_at`

* **`cards`**
  `id`, `account_id`, `card_number` (encrypted at rest), `cvv` (hashed), `expiry_date`, `is_active`, `created_at`, `updated_at`

* **`transactions`**
  `id`, `account_id`, `to_account_id` (nullable), `amount`, `type` (deposit/withdraw/transfer), `description`, `created_at`
  
---

### **5. API Endpoints:**

#### **Authentication (Public):**

* **`POST /api/v1/register`**
  Registers a new user and returns JWT + refresh token.
* **`POST /api/v1/login`**
  Authenticates a user and returns JWT + refresh token.
* **`POST /api/v1/refresh`**
  Refreshes the JWT token.

---

#### **User Management:**

* **`GET    /api/v1/users`**
  Lists all users (admin only).
* **`GET    /api/v1/users/:id`**
  Returns details for a specific user by ID.
* **`GET    /api/v1/users/me`**
  Returns the authenticated user’s profile.
* **`PUT    /api/v1/users/:id/email`**
  Updates a user’s email address.
* **`PUT    /api/v1/users/:id/password`**
  Updates a user’s password.
* **`PUT    /api/v1/users/:id/status`**
  Activates or deactivates a user account (admin only).
* **`DELETE /api/v1/users/:id`**
  Deletes a user.

---

#### **Account Management:**

* **`GET    /api/v1/accounts`**
  Lists accounts belonging to the authenticated user (admin sees all).
* **`GET    /api/v1/accounts/me`**
  Alias for the above—lists user’s own accounts.
* **`GET    /api/v1/accounts/:id`**
  Retrieves details for a specific account by ID.
* **`POST   /api/v1/accounts`**
  Creates a new bank account (max 3 accounts per user).
* **`PUT    /api/v1/accounts/:id`**
  Updates account balance (admin or owner).
* **`DELETE /api/v1/accounts/:id`**
  Deletes an account.

---

#### **Card Management:**

* **`GET    /api/v1/cards`**
  Lists all cards belonging to the authenticated user.
* **`GET    /api/v1/cards/me`**
  Alias for the above—lists user’s own cards.
* **`GET    /api/v1/cards/:id`**
  Retrieves details for a specific card by ID.
* **`POST   /api/v1/cards`**
  Issues a new card (max 3 cards per account).
* **`PUT    /api/v1/cards/:id`**
  Updates card_number, CVV, or expiry_date.
* **`PATCH  /api/v1/cards/:id`**
  Partially updates card_number, CVV, or expiry_date.
* **`PATCH  /api/v1/cards/:id/status`**
  Activates or deactivates a card.
* **`DELETE /api/v1/cards/:id`**
  Revokes a card.

  ---

#### **Transaction Management:**

* **`GET    /api/v1/transactions`**
  Lists all transactions for the authenticated user.
* **`GET    /api/v1/transactions/:id`**
  Retrieves details for a specific transaction by ID.
* **`POST   /api/v1/transactions`**
  Creates a new transaction (deposit, withdraw, transfer).
* **`PUT    /api/v1/transactions/:id`**
  Updates an existing transaction.
* **`DELETE /api/v1/transactions/:id`**
  Deletes a transaction.

---

### **6. JWT Authentication & Security:**
> Note: All protected endpoints require `Authorization: Bearer <JWT_TOKEN>`

* **Bearer Tokens:** Include in `Authorization` header.
* **Roles:**

  * **Admin:** Full access to all resources
  * **User:** Access limited to own data
* **Password Hashing:** bcrypt
* **Token Expiry & Refresh:** Configurable TTL; use `/refresh` endpoint.
* **Card Data:** PAN encrypted (AES-GCM) and masked in responses; CVV hashed and never returned.
* **User Status Update:** `/users/:id/status` is restricted to admins only.
* **Rate Limiting:** In-memory rate limiter enabled by default (20 requests/minute).

---

### **7. Project Setup & Execution:**

1. **Clone the repo:**

   ```bash
   git clone https://github.com/yusufziyrek/bank-app.git
   cd bank-app
   ```
2. **Create `.env`:**

  ```bash
  cp .env.example .env
  # Then edit .env and set real values:
  # - PG_* (database connection)
  # - JWT_SECRET (strong base64 key)
  # - CARD_ENCRYPTION_KEY (strong base64 key)
  # - REDIS_* (optional)
  ```
3. **Start test DB (Docker):**

  For local testing, use the provided example script:

  ```bash
  cd test/script
  cp test_db.example.sh test_db.sh   # Copy the example script
  # Edit test_db.sh and set your own credentials if needed
  ./test_db.sh
  ```

  > **Note:** Do not use real production passwords or secrets in this script. The example uses only dummy values. Always edit the script for your own local environment.
4. **(Optional) Start Redis cache:**

  ```bash
  docker run --name redis-local -p 6379:6379 -d redis:latest
  ```

  > Add `REDIS_ADDR=localhost:6379` to `.env`, plus `REDIS_PASSWORD` and `REDIS_DB=0` if needed. When Redis is offline the app transparently falls back to PostgreSQL only.

5. **Install & Run:**

  ```bash
  go mod tidy
  go run ./cmd
  # or
  go build -o bankapp ./cmd && ./bankapp
  ```

---

### **8. Future Improvements:**

* **Webhooks:** Notify on large transfers.
* **Two-Factor Authentication (2FA).**
* **Rate Limiting** on critical endpoints.
* **Swagger/OpenAPI** auto-docs.
* **Cache invalidation:** extend strategies (e.g., list endpoints, multi-tenant scenarios).

For full source and contributions, see the [GitHub repository](https://github.com/yusufziyrek/bank-app).
