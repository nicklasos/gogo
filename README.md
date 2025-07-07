# MyApp - Go Web API

A modern Go web API built with Echo, sqlc, and Goose migrations.

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- MySQL 8.0+
- Make

### Setup

1. **Clone and install dependencies:**
```bash
go mod tidy

go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

2. **Set up environment variables:**
```bash
cp .env.example .env
# Edit .env with your database credentials
```

3. **Create database:**
```bash
mysql -u root -p -e "CREATE DATABASE myapp;"
```

4. **Run migrations:**
```bash
# Set your DATABASE_URL environment variable first
export DATABASE_URL="root:your_password@tcp(localhost:3306)/myapp?parseTime=true"
make migrate-up
```

5. **Generate sqlc code:**
```bash
make sqlc
```

6. **Generate Swagger docs:**
```bash
make swagger
```

7. **Run the server:**
```bash
export DATABASE_URL="root:your_password@tcp(localhost:3306)/myapp?parseTime=true"
make run
```

## 📋 Available Commands

### Development
```bash
make run              # Run the server
make build            # Build the binary
make test             # Run tests
make fmt              # Format code
```

### Database
```bash
# Set DATABASE_URL environment variable first
export DATABASE_URL="root:your_password@tcp(localhost:3306)/myapp?parseTime=true"

make migrate-up       # Apply migrations
make migrate-down     # Rollback migration
make migrate-status   # Check migration status
make migrate-create   # Create new migration
make sqlc            # Generate sqlc code
```

### Documentation
```bash
make swagger         # Generate Swagger docs
```

## 🌐 API Endpoints

- `GET /api/v1/users` - List all users
- `POST /api/v1/users` - Create a user
- `GET /api/v1/users/:id` - Get user by ID
- `GET /swagger/index.html` - API documentation

## 🏗️ Project Structure

```
myapp/
├── cmd/
│   ├── api/main.go          # Main application
│   └── migrate/main.go      # Migration tool
├── internal/
│   ├── db/db.go            # Database connection
│   └── users/              # Users module
│       ├── handler.go      # HTTP handlers
│       ├── user_service.go # Business logic
│       ├── queries.sql     # SQL queries
│       ├── queries_gen.go  # Generated sqlc code
│       └── routes.go       # Route registration
├── migrations/             # Database migrations
├── docs/                   # Generated Swagger docs
├── .env.example           # Environment variables example
└── Makefile              # Development commands
```

## 🧪 Testing

```bash
make test              # Run all tests
```

## 🔧 Environment Variables

Copy `.env.example` to `.env` and configure:

- `DATABASE_URL` - MySQL connection string
- `PORT` - Server port (default: 8080)
- `APP_ENV` - Application environment
- `LOG_LEVEL` - Logging level

## 📦 Dependencies

- [Echo](https://echo.labstack.com/) - Web framework
- [sqlc](https://sqlc.dev/) - Generate type-safe Go from SQL
- [Goose](https://github.com/pressly/goose) - Database migrations
- [Swaggo](https://github.com/swaggo/swag) - Swagger documentation

## 🔐 Security Notes

- Never commit `.env` files
- Use environment variables for sensitive data
- Keep your `DATABASE_URL` secure