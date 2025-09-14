# MyApp - Go Web API

A modern Go web API built with Gin, PostgreSQL, Redis, sqlc, and Goose.

<img src="logo.jpg" alt="Logo" width="200"/>

## 🚀 Quick Start

### Prerequisites
- Go 1.23+
- PostgreSQL 13+
- Redis 6+
- Make

### Setup

1. **Clone and install dependencies:**
```bash
go mod tidy

# Install development tools
make sqlc-install
make migrate-install
make air-install
```

2. **Set up environment variables:**
```bash
cp .env.example .env
# Edit .env with your database and Redis credentials
```

3. **Run migrations:**
```bash
# Database URL is loaded automatically from .env
make migrate-up
```

4. **Generate sqlc code:**
```bash
make sqlc
```

5. **Generate Swagger docs:**
```bash
make swagger
```

6. **Run the server:**
```bash
# With hot reload (recommended for development)
make dev
```

## 🔧 Technology Stack

- **Go 1.23+** - Programming language
- **Gin** - Web framework  
- **PostgreSQL 13+** - Database
- **pgx/v5** - PostgreSQL driver
- **Redis** - Caching and session storage
- **go-redis/v9** - Redis client
- **sqlc** - Type-safe SQL code generation
- **Goose** - Database migrations
- **Air** - Hot reload for development
- **Swagger** - API documentation
- **Testify** - Testing framework

## 📋 Available Commands

### Development
```bash
make run              # Run the server
make dev              # Run with hot reload (air)
make build            # Build the binary
make test             # Run all tests
make test-unit        # Run unit tests only
make test-integration # Run integration tests only
make test-verbose     # Run tests with verbose output
make test-coverage    # Run tests with coverage
make fmt              # Format code
```

### Database
```bash
# Environment variables loaded automatically from .env
make migrate-up       # Apply migrations
make migrate-down     # Rollback migration
make migrate-status   # Check migration status
make migrate-create   # Create new migration
make sqlc             # Generate sqlc code

# Test Database
make test-migrate-up    # Migrate test db
make test-migrate-down  # Migrate rollback for test db
make test-db-setup      # Set up test database
make test-db-reset      # Reset test database
make test-with-db       # Run tests with database setup
```

### Documentation
```bash
make swagger         # Generate Swagger docs
```

### Installation
```bash
make sqlc-install    # Install sqlc CLI
make migrate-install # Install goose CLI
make air-install     # Install air CLI
```

## 🔴 Redis & Caching

The project includes a Laravel-style caching interface:

```go
// Example usage in services
var user User
err := cache.Remember(ctx, "user:123", 5*time.Minute, func() (interface{}, error) {
    return userService.GetUser(ctx, 123)
}, &user)
```

### Cache Methods
- `Get(key, dest)` - Retrieve cached value
- `Set(key, value, ttl)` - Store value with TTL
- `Remember(key, ttl, callback, dest)` - Cache-or-fetch pattern
- `Delete(key)` / `Forget(key)` - Remove from cache
- `Flush()` - Clear all cache entries
- `Has(key)` - Check if key exists

## 🌐 API Endpoints

- `GET /api/v1/users` - List all users
- `POST /api/v1/users` - Create a user
- `GET /api/v1/users/:id` - Get user by ID
- `GET /health` - Health check
- `GET /swagger/*` - API documentation

## 🏗️ Project Structure

```
myapp/
├── cmd/
│   ├── api/main.go          # Main application
│   └── migrate/main.go      # Migration tool
├── internal/
│   ├── db/db.go            # Database connection and sql files
│   ├── redis/redis.go      # Redis connection
│   ├── cache/cache.go      # Cache service
│   └── users/              # Users module
│       ├── handler.go      # HTTP handlers
│       ├── user_service.go # Business logic
│       └── routes.go       # Route registration
├── migrations/             # Database migrations
├── docs/                   # Generated Swagger docs
├── .env.example           # Environment variables example
├── .air.toml              # Air configuration
└── Makefile              # Development commands
```

## 🧪 Testing

### Laravel-Style Database Testing

This project uses a **Laravel-style testing approach** with database transaction rollback:
- **Real Database Testing** - Uses actual PostgreSQL (no mocking)
- **Transaction Rollback** - Each test runs in isolation with automatic cleanup
- **Test Database Separation** - Uses `TEST_DATABASE_URL` environment variable

### Test Structure
```
tests/
├── helpers/
│   ├── db_helper.go      # Transaction rollback helper
│   └── fixtures.go       # Test data creation
├── unit/
│   ├── news_service_test.go
│   └── feed_service_test.go
└── integration/
    └── news_api_test.go
```

### Running Tests
```bash
make test              # Run all tests
make test-unit         # Run unit tests only
make test-integration  # Run integration tests only
make test-verbose      # Run tests with verbose output
make test-coverage     # Run tests with coverage
make test-db-setup     # Set up test database
make test-db-reset     # Reset test database
make test-with-db      # Run tests with database setup
```

### Test Configuration

Add to your `.env` file:
```bash
# Test database (separate from main database)
TEST_DATABASE_URL=postgres://postgres@localhost:5432/skeleton2025_test?sslmode=disable
```

### VSCode Integration

To run tests from VSCode (clicking test icons), ensure `.vscode/settings.json` includes:
```json
{
    "go.testEnvFile": "${workspaceFolder}/.env",
    "go.testFlags": ["-v"]
}
```

This ensures VSCode loads environment variables when running tests directly.

### Test Pattern Example
```go
func TestServiceMethod(t *testing.T) {
    helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
        // Setup test data
        city := helpers.CreateTestCity(t, ctx, tx)
        
        // Test service method
        service := NewService(queries)
        result, err := service.Method(ctx, city.ID)
        
        // Assertions
        require.NoError(t, err)
        assert.Equal(t, expected, result)
    })
}
```

### Test Coverage
- **Unit Tests**: Business logic, validation, data processing
- **Integration Tests**: HTTP endpoints, request/response handling
- **Edge Cases**: Error conditions, boundary values
- **Multi-tenant**: City isolation, data separation

## 🔧 Environment Variables

Copy `.env.example` to `.env` and configure:

- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `PORT` - Server port (default: 8080)
- `APP_ENV` - Application environment
- `LOG_LEVEL` - Logging level
- `TEST_DATABASE_URL` - Test database connection string (separate from main DB)

### Example .env
```bash
DATABASE_URL=postgres://postgres@localhost:5432/skeleton2025?sslmode=disable
TEST_DATABASE_URL=postgres://postgres@localhost:5432/skeleton2025_test?sslmode=disable
REDIS_URL=redis://localhost:6379/0
PORT=8080
APP_ENV=development
LOG_LEVEL=info
```

## 🏗️ Architecture

### Core Principles
- **Clean separation of concerns** with clear layer responsibilities
- **Simplicity over complexity** - avoid unnecessary abstractions
- **Type-safe database operations** using sqlc
- **Environment-driven configuration** - no hardcoded secrets

### Layer Responsibilities
- **Handlers**: HTTP request/response, basic validation, JSON serialization
- **Services**: Business logic, input validation, complex workflows, uses sqlc directly
- **Cache**: Redis-backed caching with Laravel-style interface
- **Queries**: SQL queries managed by sqlc, type-safe database operations

## 🔐 Security Notes

- Never commit `.env` files
- Use environment variables for sensitive data
- Keep your `DATABASE_URL` and `REDIS_URL` secure
- Production-ready connection pooling configured
- SQL injection prevention via sqlc parameterized queries

## 🚀 Default Prams that should be changed in Production

You should optimize this for your needs

### Database Connection Pool (internal/db/db.go)
- Max 25 concurrent connections
- Connection recycling every 5 minutes
- Proper timeout handling

### Redis Configuration (internal/redis/redis.go)
- Connection pooling (20 max connections)
- Automatic retries with backoff
- Production timeouts and error handling

## 📖 Documentation

- **API Documentation**: Available at `/swagger/index.html` when running
- **Architecture Guide**: See `CLAUDE.md` for detailed architectural decisions
- **Development Patterns**: Consistent patterns for adding new modules

# TODO
- Place city_id right after the id column