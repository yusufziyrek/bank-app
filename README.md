# Bank App

Modern banking application API built with Go and Echo framework.

> **🚧 Project Status: Under Development**  
> This project is currently in active development. Some features may be incomplete or subject to change.

## Features

### ✅ Completed
- 🔐 JWT-based authentication with refresh token support
- 👥 User management (CRUD operations)
- 🛡️ Security measures (rate limiting, CORS, etc.)
- 📊 PostgreSQL database
- 🧪 Comprehensive test infrastructure

### 🚧 In Development
- 🏦 Account management
- 💳 Card management  
- 💰 Transaction management

## Tech Stack

- **Go 1.24.4** - Programming language
- **Echo v4** - Web framework
- **PostgreSQL** - Database
- **JWT** - Authentication
- **pgx** - PostgreSQL driver
- **Validator** - Data validation

## Quick Start

### Prerequisites

- Go 1.24.4+
- PostgreSQL 12+

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/yusufziyrek/bank-app.git
   cd bank-app
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Set up environment variables:**
   ```bash
   cp .env.example .env
   # Edit .env file with your configuration
   ```

4. **Create database:**
   ```sql
   CREATE DATABASE bankapp;
   ```

5. **Run the application:**
   ```bash
   go run cmd/main.go
   ```

## API Endpoints

### ✅ Available Endpoints

#### Authentication (Public)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/register` | User registration |
| POST | `/api/v1/login` | User login |
| POST | `/api/v1/refresh` | Refresh JWT token |

#### User Management (Protected)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users` | List all users |
| GET | `/api/v1/users/:id` | Get user details |
| PUT | `/api/v1/users/:id/email` | Update email |
| PUT | `/api/v1/users/:id/password` | Update password |
| PUT | `/api/v1/users/:id/status` | Update status |
| DELETE | `/api/v1/users/:id` | Delete user |

### 🚧 Planned Endpoints

*Account, card, and transaction management endpoints will be added as development progresses.*

## Project Structure

```
bank-app/
├── cmd/
│   └── main.go                 # Application entry point
├── common/
│   ├── app/
│   │   └── configuration.go    # Configuration management
│   └── postgresql/
│       └── postgresql.go       # Database connection
├── internal/
│   ├── controller/             # HTTP controllers
│   │   ├── dto/               # Data Transfer Objects
│   │   ├── auth_controller.go
│   │   ├── user_controller.go
│   │   └── helper.go
│   ├── model/                 # Data models
│   │   ├── user.go
│   │   ├── account.go
│   │   ├── transaction.go
│   │   └── card.go
│   ├── repository/            # Database operations
│   │   └── user_repository.go
│   ├── routes/               # Route definitions
│   │   └── routes.go
│   └── service/              # Business logic
│       └── user_service.go
├── test/                     # Test files
│   ├── infrastructure/
│   ├── service/
│   └── script/
├── go.mod
├── go.sum
└── README.md
```

## Development

### Running Tests

```bash
go test ./...
```

### Code Quality

```bash
golangci-lint run
```

### Building

```bash
go build -o bin/bank-app cmd/main.go
```

## Security

- JWT-based authentication
- Password hashing (bcrypt)
- Rate limiting
- CORS protection
- Input validation
- SQL injection protection

## Contributing

> **📝 Note:** This project is currently in active development. We welcome contributions, but please note that the codebase may undergo significant changes.

1. Fork the project
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Roadmap

- [ ] Account management features
- [ ] Card management system
- [ ] Transaction processing
- [ ] Advanced security features
- [ ] API documentation improvements
