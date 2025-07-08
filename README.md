# MyApp - Go Web API

A modern Go web API built with Echo, PostgreSQL, Redis, sqlc, and Goose migrations.

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

3. **Create database:**
```bash
createdb skeleton2025
```

4. **Run migrations:**
```bash
# Database URL is loaded automatically from .env
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
# With hot reload (recommended for development)
make dev

# Or regular run
make run
```

## 🔧 Technology Stack

- **Go 1.23+** - Programming language
- **Echo v4** - Web framework  
- **PostgreSQL 13+** - Database
- **pgx/v5** - PostgreSQL driver
- **Redis** - Caching and session storage
- **go-redis/v9** - Redis client
- **sqlc** - Type-safe SQL code generation
- **Goose** - Database migrations
- **Air** - Hot reload for development
- **Swagger** - API documentation

## 📋 Available Commands

### Development
```bash
make run              # Run the server
make dev              # Run with hot reload (air)
make build            # Build the binary
make test             # Run tests
make fmt              # Format code
```

### Database
```bash
# Environment variables loaded automatically from .env
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
│   ├── db/db.go            # Database connection
│   ├── redis/redis.go      # Redis connection
│   ├── cache/cache.go      # Cache service
│   └── users/              # Users module
│       ├── handler.go      # HTTP handlers
│       ├── user_service.go # Business logic
│       ├── queries.sql     # SQL queries
│       ├── queries_gen.go  # Generated sqlc code
│       └── routes.go       # Route registration
├── migrations/             # Database migrations
├── docs/                   # Generated Swagger docs
├── .env.example           # Environment variables example
├── .air.toml              # Air configuration
└── Makefile              # Development commands
```

## 🧪 Testing

```bash
make test              # Run all tests
```

## 🔧 Environment Variables

Copy `.env.example` to `.env` and configure:

- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `PORT` - Server port (default: 8080)
- `APP_ENV` - Application environment
- `LOG_LEVEL` - Logging level

### Example .env
```bash
DATABASE_URL=postgres://postgres@localhost:5432/skeleton2025?sslmode=disable
REDIS_URL=redis://localhost:6379/0
PORT=8080
APP_ENV=development
LOG_LEVEL=info
```

## 📦 Dependencies

- [Echo](https://echo.labstack.com/) - Web framework
- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver
- [go-redis](https://github.com/redis/go-redis) - Redis client
- [sqlc](https://sqlc.dev/) - Generate type-safe Go from SQL
- [Goose](https://github.com/pressly/goose) - Database migrations
- [Air](https://github.com/cosmtrek/air) - Hot reload
- [Swaggo](https://github.com/swaggo/swag) - Swagger documentation

## 🏗️ Architecture

### Core Principles
- **Clean separation of concerns** with clear layer responsibilities
- **Simplicity over complexity** - avoid unnecessary abstractions
- **Type-safe database operations** using sqlc
- **Environment-driven configuration** - no hardcoded secrets

### Request Flow
```
HTTP Request → Handler → Service → sqlc → Database
                    ↘ Cache (Redis)
```

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

## 🚀 Production Ready

The project includes production-ready configurations:

### Database Connection Pool
- Max 25 concurrent connections
- Connection recycling every 5 minutes
- Proper timeout handling

### Redis Configuration
- Connection pooling (20 max connections)
- Automatic retries with backoff
- Production timeouts and error handling

## 📖 Documentation

- **API Documentation**: Available at `/swagger/index.html` when running
- **Architecture Guide**: See `CLAUDE.md` for detailed architectural decisions
- **Development Patterns**: Consistent patterns for adding new modules