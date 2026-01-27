# Claude Project Memory

## Architecture

### Core Philosophy
- **Clean separation of concerns** with clear layer responsibilities
- **Simplicity over complexity** - avoid unnecessary abstractions
- **Type-safe database operations** using sqlc
- **Environment-driven configuration** - no hardcoded secrets

### Directory Structure
```
gogo/
├── cmd/api/main.go              # Main application entry
├── internal/
│   ├── app.go                   # App context with DB, Cache, Logger
│   ├── auth/                    # Authentication module
│   │   ├── auth_service.go      # Business logic
│   │   ├── handlers.go          # HTTP handlers
│   │   ├── routes.go            # Route registration
│   │   └── types.go             # Request/response types
│   ├── example/                 # Example CRUD module
│   │   ├── example_service.go   # Business logic
│   │   ├── handler.go           # HTTP handlers
│   │   ├── routes.go            # Route registration
│   │   └── types.go             # Request/response types
│   ├── db/
│   │   └── queries/              # SQL queries
│   ├── middleware/
│   │   ├── user_auth.go         # JWT authentication
│   │   └── pagination.go        # Pagination context
│   └── responses.go             # PaginationMeta helper
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
- JWT - Authentication
- Testify - Testing framework

## Module Pattern
When adding new modules:

```go
// internal/orders/
├── handler.go           # HTTP endpoints - receives only needed services
├── order_service.go     # Business logic
├── routes.go           # Route registration - receives *internal.App, handles DI
└── types.go            # All request/response types
```

### Dependency Injection Pattern
- **Routes** (`routes.go`): Only layer that knows about `*internal.App`
- **Handlers**: Receive specific services they need (e.g., `*OrderService`)
- **Services**: Receive specific dependencies (e.g., `*db.Queries`, logger, cache)

## Context Patterns

### User ID from Context
Use `middleware.GetUserIDFromContext(c)` in handlers to get authenticated user ID:

```go
func (h *Handler) CreateOrder(c *gin.Context) {
    userID, err := middleware.GetUserIDFromContext(c)
    if err != nil {
        errs.RespondWithUnauthorized(c, "Unauthorized")
        return
    }
    // Use userID...
}
```

### Pagination from Context
Use `middleware.GetPaginationParamsFromContext(c, default, min, max)` in handlers:

```go
func (h *Handler) ListOrders(c *gin.Context) {
    pagination, err := middleware.GetPaginationParamsFromContext(c, 20, 1, 100)
    if err != nil {
        errs.RespondWithBadRequest(c, errs.ErrKeyBadRequest, err.Error())
        return
    }
    // Use pagination.Page and pagination.PageSize...
}
```

## Types.go Pattern

### Rule
- **All request/response types** go in `types.go` within each module
- **Service types** (e.g., `PaginatedExamplesResult`) are defined in service files
- **Handlers** use types from `types.go` for requests/responses
- **Services** use internal types and convert to handler types
- Clear separation: handlers know about types.go, services don't

### Example
```go
// types.go
type CreateRequest struct {
    Title string `json:"title" binding:"required"`
}

type ExampleResponse struct {
    ID    int32  `json:"id"`
    Title string `json:"title"`
}

// example_service.go
type PaginatedExamplesResult struct {
    Data     []db.Example
    Total    int64
    Page     int32
    PageSize int32
}

// handler.go
func (h *Handler) Create(c *gin.Context) {
    var req CreateRequest  // From types.go
    // ...
    response := ExampleResponse{...}  // From types.go
}
```

## Database Management

### Migration Creation Process
```bash
# Create migrations manually with sequential numbering
# Format: migrations/001_description.sql, 002_description.sql, etc.

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

## Development Commands
```bash
# Development
make run              # Start server
make build            # Build binary
make test             # Run all tests

# Database
make migrate-up       # Apply migrations
make sqlc            # Generate sqlc code
make swagger         # Generate API docs
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
7. **Context patterns** - Use middleware for user ID and pagination
8. **Types.go pattern** - All request/response types in types.go

## Testing Framework

### Laravel-Style Database Testing
- **Transaction Rollback Pattern** - Each test runs in isolation with automatic rollback
- **Real Database Testing** - Uses actual PostgreSQL (no mocking)
- **Test Database Separation** - Uses `TEST_DATABASE_URL` environment variable

### Test Patterns
```go
func TestServiceMethod(t *testing.T) {
    helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
        // Setup test data
        example := helpers.CreateTestExample(t, ctx, tx)
        
        // Test service method
        service := NewService(queries)
        result, err := service.Method(ctx, example.ID)
        
        // Assertions
        require.NoError(t, err)
        assert.Equal(t, expected, result)
    })
}
```

## What We DON't Use
- NO Repository Pattern - Services use sqlc directly
- NO ORM - Raw SQL with sqlc for type safety
- NO complex abstractions - Keep it simple
- NO Test Mocking - Real database with transaction rollback
