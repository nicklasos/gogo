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
├── cmd/api/main.go              # Main application entry (--port, --test-db)
├── cmd/cron/main.go             # Standalone scheduler
├── cmd/cli/main.go              # CLI (migrate, smoke tests)
├── internal/
│   ├── app.go                   # App context with DB, Cache, Logger, Images
│   ├── images.go                # Relative path → public URL helper
│   ├── auth/                    # Authentication module
│   │   ├── auth_service.go      # Business logic
│   │   ├── handlers.go          # HTTP handlers
│   │   ├── routes.go            # Route registration (returns *AuthService)
│   │   └── types.go             # Request/response types
│   ├── example/                 # Example CRUD module (cache demo)
│   ├── uploads/                 # File uploads + public static route
│   ├── db/
│   │   └── queries/             # SQL queries (incl. technical Healthcheck)
│   ├── middleware/
│   │   ├── user_auth.go         # JWT authentication
│   │   └── pagination.go        # Page + cursor pagination
│   ├── cache/                   # RedisCache + MemoryCache
│   ├── errs/                    # Domain errors + WrapDatabaseError
│   ├── scheduler/               # Cron (cleanup refresh tokens)
│   └── responses.go             # PaginationMeta helper
├── migrations/                  # Goose database migrations
└── Makefile                     # Development commands (make help)
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
- **Auth**: `auth.RegisterRoutes(app)` returns `*AuthService` for other modules

## Context Patterns

### User ID from Context
Use `middleware.GetUserIDFromContext(c)` in handlers to get authenticated user ID.

### Pagination from Context
- Page-based: `middleware.GetPaginationParamsFromContext(c, default, min, max)`
- Cursor by ID: `middleware.GetLastIDPaginationParamsFromContext(c, default, min, max)`
- Cursor by time: `middleware.GetCreatedAtPaginationParamsFromContext(c, default, min, max)`

## Types.go Pattern

### Rule
- **All request/response types** go in `types.go` within each module
- **Service types** (e.g., `PaginatedExamplesResult`) are defined in service files
- **Handlers** use types from `types.go` for requests/responses
- **Services** use internal types and convert to handler types

## Database Management

### Migration Creation Process
```bash
# Create migrations manually with sequential numbering
# Format: migrations/001_description.sql, 002_description.sql, etc.

# Apply migrations
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
make help             # List targets
make run              # Start server
make run-test-db      # Start against TEST_DATABASE_URL
make build            # Build binary
make test             # Run all tests (auto-migrates test DB)
make migrate-up       # Apply migrations
make sqlc             # Generate sqlc code
make swagger          # Generate API docs
```

## Code Conventions
- **Handlers**: `GetUser`, `CreateUser`, `ListUsers`
- **Services**: `UserService`, `OrderService`
- **SQL queries**: `GetUserByID`, `CreateUser`, `ListUsers`
- **Files**: `user_service.go`, `order_handler.go`
- **Cache keys**: `user:123`, `examples:user:123:page:1:size:20`

## Key Principles
1. **Dependency Injection via Routes** - Only `routes.go` knows about `*internal.App`
2. **Handlers receive specific services** - No direct access to `*internal.App`
3. **Services own business logic** - Keep handlers thin
4. **Use sqlc directly** - No repository abstraction
5. **Environment-driven config** - No hardcoded values
6. **Module-based organization** - Self-contained domains
7. **Context patterns** - Use middleware for user ID and pagination
8. **Types.go pattern** - All request/response types in types.go
9. **Cache where useful** - `Remember` + invalidate on writes (see example module)

## Testing Framework

### Laravel-Style Database Testing
- **Transaction Rollback Pattern** - Each test runs in isolation with automatic rollback
- **Real Database Testing** - Uses actual PostgreSQL (no mocking)
- **Test Database Separation** - Uses `TEST_DATABASE_URL` environment variable
- **MemoryCache** - Tests use in-memory cache (no Redis required)
- **GenerateTestJWT** - Helper for authenticated integration requests

### Test Patterns
```go
func TestServiceMethod(t *testing.T) {
    helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
        example := helpers.CreateTestExample(t, ctx, tx, user.ID)

        service := NewService(queries, nil)
        result, err := service.Method(ctx, example.ID)

        require.NoError(t, err)
        assert.Equal(t, expected, result)
    })
}
```

## What We DON'T Use
- NO Repository Pattern - Services use sqlc directly
- NO ORM - Raw SQL with sqlc for type safety
- NO complex abstractions - Keep it simple
- NO Test Mocking of DB - Real database with transaction rollback
