# Claude Project Memory

## Architecture

### Core Philosophy
- **Clean separation of concerns** with clear layer responsibilities
- **Simplicity over complexity** - avoid unnecessary abstractions
- **Type-safe database operations** using sqlc
- **Environment-driven configuration** - no hardcoded secrets

### Directory Structure
```
myapp/
├── cmd/api/main.go              # Main application entry
├── internal/
│   ├── db/db.go                 # Database connection
│   └── users/                   # Domain modules
│       ├── handler.go           # HTTP controllers
│       ├── user_service.go      # Business logic
│       └── routes.go            # Route registration
├── migrations/                  # Goose database migrations
└── Makefile                     # Development commands
```

### Layer Responsibilities
- **Routes**: Dependency injection, receives `*internal.App` and creates services/handlers with specific dependencies
- **Handlers**: HTTP request/response, basic validation, JSON serialization, receives only needed services
- **Services**: Business logic, input validation, complex workflows, uses sqlc directly
- **Queries**: SQL queries managed by sqlc, type-safe database operations

## Technology Stack
- Go 1.24+
- Gin
- PostgreSQL 15
- pgx/v5
- Redis
- go-redis/v9 - Redis client
- sqlc - Type-safe SQL code generation
- Goose - Database migrations
- Swaggo - Swagger documentation
- Testify - Testing framework for unit and integration tests

## Module Pattern
When adding new modules (orders, products, etc.):

```go
// internal/orders/
├── handler.go           # HTTP endpoints - receives only needed services
├── order_service.go     # Business logic
└── routes.go           # Route registration - receives *internal.App, handles DI
```

### Dependency Injection Pattern
- **Routes** (`routes.go`): Only layer that knows about `*internal.App`
- **Handlers**: Receive specific services they need (e.g., `*OrderService`)
- **Services**: Receive specific dependencies (e.g., `*db.Queries`, logger, cache)

```go
// Example: internal/orders/routes.go
func RegisterRoutes(app *internal.App) {
    // Create service with only needed dependencies
    service := NewOrderService(app.Queries, app.Logger)
    
    // Create handler with only needed services
    handler := NewHandler(service)
    
    app.Api.POST("/orders", handler.CreateOrder)
    app.Api.GET("/orders", handler.ListOrders)
}

// Example: internal/orders/handler.go
func NewHandler(service *OrderService) *Handler {
    return &Handler{service: service}
}
```

## Database Management

### Migration Creation Process
```bash
# Create migrations manually with sequential numbering
# Format: migrations/001_description.sql, 002_description.sql, etc.
# 
# Migration template:
# -- +goose Up
# -- +goose StatementBegin
# CREATE TABLE example (
#     id SERIAL PRIMARY KEY,
#     name VARCHAR(255) NOT NULL
# );
# -- +goose StatementEnd
# 
# -- +goose Down
# -- +goose StatementBegin
# DROP TABLE IF EXISTS example;
# -- +goose StatementEnd

# Apply migrations
export DATABASE_URL="your_connection_string"
make migrate-up

# Generate sqlc after schema changes
make sqlc
```

### Migration Naming Convention
- **Format**: `001_description.sql`, `002_description.sql`, etc.
- **Location**: `migrations/` directory
- **Always include timestamps**: `created_at`, `updated_at` with `DEFAULT CURRENT_TIMESTAMP`
- **Use soft deletes**: `deleted_at` 

## Development Commands
```bash
# Development
make run              # Start server
make build            # Build binary
make test             # Run all tests
make test-unit        # Run unit tests only
make test-integration # Run integration tests only

# Database
make migrate-up       # Apply migrations
make sqlc            # Generate sqlc code
make swagger         # Generate API docs

# Test Database
make test-db-setup    # Set up test database
make test-db-reset    # Reset test database
make test-with-db     # Run tests with database setup
```

## Code Conventions
- **Handlers**: `GetUser`, `CreateUser`, `ListUsers`
- **Services**: `UserService`, `OrderService`
- **SQL queries**: `GetUserByID`, `CreateUser`, `ListUsers`
- **Files**: `user_service.go`, `order_handler.go`
- **Cache keys**: `user:123`, `posts:user:123`

## Key Principles
1. **Dependency Injection via Routes** - Only `routes.go` knows about `*internal.App`
2. **Handlers receive specific services** - No direct access to `*internal.App`
3. **Services own business logic** - Keep handlers thin
4. **Use sqlc directly** - No repository abstraction
5. **Environment-driven config** - No hardcoded values
6. **Module-based organization** - Self-contained domains
7. **Real database testing** - Transaction rollback for isolation
8. **Test-driven development** - Comprehensive unit and integration tests

## Testing Framework

### Laravel-Style Database Testing
- **Transaction Rollback Pattern** - Each test runs in isolation with automatic rollback
- **Real Database Testing** - Uses actual PostgreSQL (no mocking)
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

### Test Patterns
```go
// Transaction rollback pattern
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

### Test Configuration
- **Environment Required**: `TEST_DATABASE_URL` must be set
- **Panic on Missing Config**: Tests panic if TEST_DATABASE_URL is not found
- **Automatic Cleanup**: Each test runs in transaction that gets rolled back
- **Isolation**: Tests don't affect each other or production data

### VSCode Test Integration
To run tests from VSCode (clicking test icons), ensure `.vscode/settings.json` includes:
```json
{
    "go.testEnvFile": "${workspaceFolder}/.env",
    "go.testFlags": ["-v"]
}
```

This ensures VSCode loads environment variables when running tests directly.

### Test Coverage Examples
- **Unit Tests**: Business logic, validation, data processing
- **Integration Tests**: HTTP endpoints, request/response handling
- **Edge Cases**: Error conditions, boundary values, constraint violations
- **Multi-tenant**: City isolation, data separation

### Running Tests
```bash
# All tests (requires TEST_DATABASE_URL in .env)
make test

# Unit tests only
make test-unit

# Integration tests only  
make test-integration

# With verbose output
make test-verbose

# With coverage
make test-coverage

# Set up test database first
make test-db-setup
```

## What We DON'T Use
- NO Repository Pattern - Services use sqlc directly
- NO ORM - Raw SQL with sqlc for type safety
- NO complex abstractions - Keep it simple
- NO Test Mocking - Real database with transaction rollback