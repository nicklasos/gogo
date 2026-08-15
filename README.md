# GOGO – Go API Template with SQLC

A production-ready Go API template built with **Gin**, **PostgreSQL**, **Redis**, **SQLC**, and **Goose** migrations.

## Quick Start

### Prerequisites
- Go 1.24+, PostgreSQL 13+, Redis 6+, Make

### Setup
```bash
# Install dependencies and tools
go mod tidy
make sqlc-install migrate-install air-install

# Configure environment
cp .env.example .env  # Edit with your credentials (JWT_SECRET required)

# Setup database
make migrate-up
make sqlc

# Start development server
make dev
```

## Features

- **Authentication**: JWT auth with registration, login, refresh tokens, and logout (revokes refresh tokens)
- **CRUD Example**: Complete example module with Redis `Remember` caching
- **Pagination**: Page-based and cursor helpers (`last_id`, `last_created_at`)
- **Type Safety**: SQLC for type-safe database operations
- **Swagger**: Auto-generated API documentation
- **Uploads**: File upload with list/get/delete and static file serving
- **Healthcheck**: `/health` pings PostgreSQL
- **Scheduler**: Optional cron jobs (refresh-token cleanup)

## Tech Stack

- **Go 1.24+** + **Gin** - API framework
- **PostgreSQL** + **pgx/v5** - Database with connection pooling
- **SQLC** - Type-safe SQL code generation
- **Redis** + **go-redis/v9** - Caching
- **Goose** - Database migrations
- **JWT** - Authentication
- **Swaggo** - API documentation

## Project Structure

```
gogo/
├── cmd/api/main.go              # Application entry point (--port, --test-db)
├── cmd/cron/main.go             # Standalone scheduler
├── cmd/cli/main.go              # CLI (migrate, test)
├── internal/
│   ├── app.go                   # App context with DB, Cache, Logger, Images
│   ├── auth/                    # Authentication module
│   ├── example/                 # Example CRUD module (cache demo)
│   ├── uploads/                 # File uploads
│   ├── db/queries/              # SQL queries (sqlc)
│   ├── middleware/              # JWT, pagination, recovery
│   ├── cache/                   # Redis + MemoryCache
│   ├── errs/                    # Domain errors
│   └── scheduler/               # Cron jobs
├── migrations/                  # Goose database migrations
└── sqlc.yaml
```

## Development Commands

```bash
make help             # List all targets

# Development
make dev              # Hot reload server
make run              # Start API
make run-test-db      # Start API against TEST_DATABASE_URL
make test             # Run all tests (auto-migrates test DB)

# Database & SQLC
make migrate-up       # Apply migrations
make sqlc             # Generate SQLC code (run after SQL changes!)

# Documentation
make swagger          # Generate API docs
```

## API Endpoints

### Auth
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/refresh` - Refresh token
- `GET /api/v1/auth/me` - Get current user (protected)
- `POST /api/v1/auth/logout` - Logout and revoke refresh tokens (protected)

### Examples
- `GET /api/v1/examples` - List examples with pagination (protected, cached)
- `POST /api/v1/examples` - Create example (protected)
- `GET /api/v1/examples/:id` - Get example (protected)
- `PUT /api/v1/examples/:id` - Update example (protected)
- `DELETE /api/v1/examples/:id` - Delete example (protected)

### Uploads
- `POST /api/v1/uploads` - Upload a file (protected)
- `GET /api/v1/uploads` - List uploads (protected)
- `GET /api/v1/uploads/:id` - Get upload (protected)
- `DELETE /api/v1/uploads/:id` - Delete upload (protected)
- `GET /api/files/*` - Serve uploaded files (public)

### Other
- `GET /health` - Health check (DB ping)
- `GET /api/v1/health` - Same health check under API prefix
- `GET /swagger/*` - API documentation

## Environment Variables

```bash
DATABASE_URL=postgres://postgres@localhost:5432/gogo?sslmode=disable
TEST_DATABASE_URL=postgres://postgres@localhost:5432/gogo_test?sslmode=disable
REDIS_URL=redis://localhost:6379/0
JWT_SECRET=your-secret-key-here
PORT=8181
APP_ENV=development
LOG_LEVEL=info
ENABLE_SCHEDULER=false
```

## Patterns

### Context Pattern
- **User ID**: Use `middleware.GetUserIDFromContext(c)` in handlers
- **Pagination**: Use `middleware.GetPaginationParamsFromContext(c, default, min, max)`
- **Cursor pagination**: `GetLastIDPaginationParamsFromContext` / `GetCreatedAtPaginationParamsFromContext`

### Types.go Pattern
- All request/response types go in `types.go` within each module
- Services define internal types (e.g., `PaginatedExamplesResult`) in service files
- Handlers convert service types to response types from `types.go`

### Cache Pattern
- Use `cache.Remember` for short-lived list responses (see `example` module)
- Use `cache.MemoryCache` in tests (no Redis required)
- Invalidate on create/update/delete

## Uploads Module

The uploads module allows users to upload files (images, videos, documents, audio) and stores metadata in the database. Files are served from `/api/files/...`.

### Configuration

```go
config := &uploads.UploadConfig{
    UploadFolder: "./uploads",
    BaseURL:      "http://localhost:8181/api/files",
    MaxFileSize:  50 * 1024 * 1024, // 50MB
    AllowedTypes: []string{".jpg", ".png", ".pdf"},
    GetFolderID: func(ctx context.Context, userID int32) (int32, error) {
        return userID, nil
    },
}
```

## Adding New Modules

1. Create migration: `migrations/XXX_create_table.sql`
2. Write SQL queries in `internal/db/queries/module.sql`
3. Generate code: `make sqlc`
4. Create module: `internal/module/{service,handler,routes,types}.go`
5. Register routes in `cmd/api/main.go`

## Error Handling

The project uses structured error handling with the `errs` package. See `docs/ERRORS.md` for complete guide and examples.

**Quick reference:**
- Services return domain errors: `errs.NewNotFoundError(key, message)`
- Handlers use: `errs.RespondWithError(c, err)` or `errs.RespondWithValidationError(c, err)`
- DB helpers: `errs.WrapDatabaseError`, `errs.DomainErrorFromPostgresUniqueViolation`
- See `internal/example/` for complete examples

## Links

- **API Docs**: `/swagger/index.html` when running
- **Error Handling**: See `docs/ERRORS.md` for error handling guide
- **Architecture**: See `CLAUDE.md` for detailed patterns
- **Deployment**: See `DEPLOYMENT.md` for supervisord setup
