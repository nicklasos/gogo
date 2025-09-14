# GOGO – Go API Template with SQLC

A production-ready Go API template built with **Gin**, **PostgreSQL**, **Redis**, **SQLC**, and **Goose** migrations.

![Logo](logo.jpg)

## 🚀 Quick Start

### Prerequisites
- Go 1.24+, PostgreSQL 13+, Redis 6+, Make

### Setup
```bash
# Install dependencies and tools
go mod tidy
make sqlc-install migrate-install air-install

# Configure environment
cp .env.example .env  # Edit with your credentials

# Setup database
make migrate-up
make sqlc # sqlc generate

# Start development server
make dev
```

## 🔧 Tech Stack
- **Go 1.24+** + **Gin** - API framework
- **PostgreSQL** + **pgx/v5** - Database with connection pooling
- **SQLC** - Type-safe SQL code generation
- **Redis** + **go-redis/v9** - Caching
- **Goose** - Database migrations
- **Air** - Hot reload development

## 🗃️ SQLC Integration

Write SQL queries in `internal/db/queries/*.sql`, generate type-safe Go code:

**SQL** (`internal/db/queries/cities.sql`):
```sql
-- name: GetCityByID :one
SELECT * FROM cities WHERE id = $1 LIMIT 1;

-- name: ListCities :many
SELECT * FROM cities ORDER BY name ASC;
```

**Generated Usage**:
```go
// In service layer
cities, err := s.queries.ListCities(ctx)  // Type-safe, no ORM
```

**Configuration** (`sqlc.yaml`):
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/db/queries"
    schema: "migrations/"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        emit_db_tags: true
        emit_json_tags: true
```

## 📋 Development Commands

```bash
# Development
make dev              # Hot reload server
make test             # Run all tests
make test-coverage    # Tests with coverage

# Database & SQLC
make migrate-up       # Apply migrations
make migrate-create   # Create new migration
make sqlc             # Generate SQLC code (run after SQL changes!)

# Documentation
make swagger          # Generate API docs
```

## 🏗️ Project Structure

```
gogo/
├── cmd/api/main.go              # Application entry point
├── internal/
│   ├── app.go                   # App context with DB, Cache, Logger
│   ├── db/
│   │   ├── queries/cities.sql   # Raw SQL queries
│   │   ├── cities.sql.go        # SQLC generated code
│   │   └── models.go            # SQLC generated models
│   ├── cities/                  # Domain module
│   │   ├── handler.go           # HTTP handlers
│   │   ├── cities_service.go    # Business logic using SQLC
│   │   └── routes.go            # Route registration
│   ├── cache/cache.go           # Redis caching (Laravel-style)
│   └── middleware/              # Custom middleware
├── migrations/                  # Goose database migrations
├── tests/                       # Unit & integration tests
└── sqlc.yaml                    # SQLC configuration
```

## 🔴 Caching

Laravel-style Redis caching interface:

```go
// Cache-or-fetch pattern
var city City
err := cache.Remember(ctx, "city:123", 5*time.Minute, func() (interface{}, error) {
    return citiesService.GetCityByID(ctx, 123)
}, &city)
```

## 🧪 Testing

Laravel-style testing with real database and transaction rollback:

```go
func TestCitiesService_ListCities(t *testing.T) {
    helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
        // Setup test data using SQLC
        city := helpers.CreateTestCity(t, ctx, tx)
        
        // Test service
        service := cities.NewCitiesService(queries)
        result, err := service.ListCities(ctx)
        
        require.NoError(t, err)
        assert.Len(t, result, 1)
    })
}
```

**Test Commands**:
```bash
make test-db-setup    # Setup test database
make test-with-db     # Run tests with DB
```

## 🔧 Environment Variables

```bash
# .env example
DATABASE_URL=postgres://postgres@localhost:5432/gogo?sslmode=disable
TEST_DATABASE_URL=postgres://postgres@localhost:5432/gogo_test?sslmode=disable
REDIS_URL=redis://localhost:6379/0
PORT=8080
APP_ENV=development
LOG_LEVEL=info
```

## 🎯 Adding New Modules

1. **Create migration**: `make migrate-create name=create_users_table`
2. **Write SQL queries** in `internal/db/queries/users.sql`
3. **Generate code**: `make sqlc`
4. **Create module**: `internal/users/{handler,service,routes}.go`
5. **Register routes** in `cmd/api/main.go`

## 🌐 API Endpoints

- `GET /api/v1/cities` - List cities
- `GET /health` - Health check  
- `GET /swagger/*` - API documentation

## 📖 Links

- **API Docs**: `/swagger/index.html` when running
- **SQLC Docs**: [docs.sqlc.dev](https://docs.sqlc.dev/)
- **Architecture**: See `CLAUDE.md` for detailed decisions