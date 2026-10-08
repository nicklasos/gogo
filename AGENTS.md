# Gogo – Agent Guide

## Start here
- **Recipes**: `docs/recipes/` has step-by-step procedures for the common changes: add a module, add an endpoint, change the schema, add a scheduled job. Use the matching one instead of improvising.
- **Reference module**: `internal/example`. Copy its shape.
- **Finish every task with `make test`** and fix what fails. Do not start the API; the owner runs it.
- **The kit**: this repo is the API. [gogo-front](https://github.com/nicklasos/gogo-front) is its admin UI and generates its TypeScript types from `docs/swagger.json`, so run `make swagger` after changing request or response types. [gopulse](https://github.com/nicklasos/gopulse) is the monitoring dashboard mounted at `/_pulse`.

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
├── cmd/cli/main.go              # CLI (migrate, create-user, mail, smoke test); global --test flag
├── config/
│   ├── config.go                # Env → Config
│   └── testdb.go                # Test-database safety guard
├── internal/
│   ├── app.go                   # App context with DB, Tx, Cache, Logger, Mail, AuthMiddleware
│   ├── server/server.go         # NewEngine (middleware) + RegisterRoutes (the module list)
│   ├── auth/                    # Authentication module
│   │   ├── auth_service.go      # Business logic
│   │   ├── handlers.go          # HTTP handlers
│   │   ├── routes.go            # Route registration (sets app.AuthMiddleware)
│   │   └── types.go             # Request/response types
│   ├── users/                   # User management (super admins, admins, users)
│   ├── example/                 # Example CRUD module (cache demo)
│   ├── uploads/                 # File uploads + public static route
│   ├── health/                  # /health and /api/v1/health
│   ├── db/
│   │   ├── tx.go                # TxRunner.WithTx transaction helper
│   │   └── queries/             # SQL queries (incl. technical Healthcheck)
│   ├── middleware/
│   │   ├── user_auth.go         # JWT authentication, loads roles from DB
│   │   ├── require_role.go      # Roles + RequireRole / RequireAnyRole
│   │   ├── cors.go              # CORS from CORS_ALLOWED_ORIGINS
│   │   ├── logging.go           # RequestID, Recovery, ErrorHandler
│   │   ├── request.go           # CurrentUserID, PathID, Page: handler helpers that answer the error themselves
│   │   └── pagination.go        # ?page / ?page_size parsing
│   ├── cache/                   # RedisCache + MemoryCache
│   ├── mail/                    # SMTP Service + MemorySender (tests)
│   ├── monitoring/              # gopulse dashboard wiring (nil-safe helpers)
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
- Go 1.25+
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
- **Auth**: `auth.RegisterRoutes(app)` sets `app.AuthMiddleware`; protect a route group with `group.Use(app.AuthMiddleware)`
- **Registration**: every module is listed once in `internal/server/server.go` `RegisterRoutes`. The API binary and the test server both call it, so a new module is one line there.

## Roles
- Roles live in `users.roles` (`TEXT[]`): `super-admin`, `admin`, `user` (constants in `middleware/require_role.go`).
- `UserAuthMiddleware` loads roles from the database on every request; never read roles from JWT claims.
- Guard routes with `middleware.RequireRole(middleware.RoleAdmin)` or `RequireAnyRole(...)`. A super admin passes every role check.
- In handlers use `middleware.GetUserRolesFromContext(c)`.
- Who manages whom is one rule, `users.CanManage`: super admins manage everyone, admins manage only accounts whose every role is `user`.
- The first super admin is created with `make cli-create-user EMAIL=... PASSWORD=...`.

## Policies
- Who may do what to a record lives in the module's `policy.go` as pure functions: `CanView(actor middleware.Actor, record db.X) bool`.
- A handler gets the actor with `middleware.CurrentActor(c)` and passes it to the service. The service loads the record, calls the policy, and returns a 403 domain error on a no (404 stays for "does not exist").
- Look records up by ID alone and let the policy decide; do not hide the rule inside a `WHERE user_id = ...`.
- `middleware.RequireRole` on a route group is for "this whole area is admins only". A policy is for rules that depend on the record.
- gogo-front mirrors each policy in `features/<module>/policy.ts` to hide what is not allowed. The API stays the authority.
- `internal/example/policy.go` and `internal/users/policy.go` are the references; `tests/unit/policy_test.go` shows how to test one.

## Factories and seed data
- `internal/factory` creates records with defaults: `factory.User(t, tx)`, `factory.User(t, tx, factory.WithRoles("admin"), factory.Unverified())`, `factory.Example(t, tx, user.ID, factory.WithTitle("..."))`, `factory.Examples(t, tx, user.ID, 25, "Sample")`, `factory.Upload(t, tx, user.ID)`.
- Every new table gets a factory. Tests never write `INSERT` statements by hand.
- Users from a factory have the password `factory.DefaultPassword` and a verified email unless `factory.Unverified()` is passed.
- `make seed` fills an empty development database through the same factories (`cmd/cli/internal/commands/seed.go`); add a new module's sample data there.

## Transactions
Services that need more than one write take `*db.TxRunner` (from `app.Tx`) and use:
```go
err := s.tx.WithTx(ctx, func(q *db.Queries) error {
    // use q, not s.queries; return an error to roll back
})
```
In tests the runner is built on the test transaction, so `WithTx` becomes a savepoint.

## Auth
- Settings come from `config` and reach the module as `auth.Options` in `auth/routes.go`.
- Emailed links (password reset, email verification) are rows in `auth_tokens`: single-use, expiring, stored as SHA-256 hashes. Issue with `issueEmailToken`, redeem with `consumeEmailToken` inside `WithTx`.
- `forgot-password` must never reveal whether an email exists: no error, no different status.
- `middleware.AuthRateLimit` guards login (call `middleware.MarkAuthSuccess(c)` on success); `middleware.RateLimit(cache, log, scope, limit, window)` is the generic per-IP limiter for anything else.
- A 401 from an authenticated route makes clients try a token refresh, so use 400 or 403 for anything that is not "this session is invalid".
- Registration is a 403 unless `ALLOW_REGISTRATION=true`.
- Pass every email that enters the system through `internal.NormalizeEmail` before storing or looking it up.
- Refresh tokens and emailed tokens are stored as hashes (`hashToken`); the raw value exists only in the response or the email.
- `/health` reports the database and any cache that implements `cache.Pinger`. Add a check there for each new dependency the API cannot serve without.

## Uploads
- Go through `UploadService`; file bytes go through the `Storage` interface (`LocalStorage` by default). Never build disk paths from request data yourself: `LocalStorage.Resolve` is the traversal guard.
- New allowed extensions go in `DefaultUploadConfig`; `matchesContent` decides what content an extension must have.

## Monitoring
- gopulse (`github.com/nicklasos/gopulse`) is wired in `cmd/api/main.go` through `internal/monitoring`. `app.Pulse` is nil when `PULSE_PASSWORD` is unset, and it is nil in tests.
- Never call `app.Pulse` methods without a nil check; `monitoring.Record(app.Pulse, metric, key, value)` does the check for custom metrics.
- Queries and logs are linked to their request only when they are given the request context: `c.Request.Context()` into services, `logger.InfoContext(ctx, ...)` for logs.
- When something is slow or failing locally, the dashboard at `/_pulse` shows which route, which query and which log lines.

## Mail
- Services take a `mail.Sender` (from `app.Mail`) and call `Send(ctx, mail.Message{To, Subject, Text, HTML})`.
- `Send` only logs when `APP_DEBUG=true`; `mail.Service.SendLive` always dials SMTP (used by `make cli-mail TO=...`).
- Tests get a `mail.MemorySender` from `CreateTestServer`; assert on `server.Mail.Sent()`. Never dial SMTP from a test.
- Send after the database work has committed, not inside `WithTx`.

## Context Patterns

### User ID from Context
`userID, ok := middleware.CurrentUserID(c)` in a handler; when `ok` is false it has already answered 401, so just return.

### Request ID
`middleware.RequestID` puts an ID on the request context and the `X-Request-ID` response header. Log with the `*Context` methods (`logger.ErrorContext(c.Request.Context(), ...)`) and the ID is added to the record automatically.

### Pagination from Context
`page, ok := middleware.Page(c)` gives `page.Page`, `page.PageSize` and `page.Offset()` with the standard limits (20 by default, 100 at most). `middleware.GetPaginationParamsFromContext(c, default, min, max)` is there for an endpoint that needs other limits.

### Path parameters
`id, ok := middleware.PathID(c, "id")` parses a positive integer and answers 400 otherwise.

### Handler shape
Keep handlers to: helpers above, bind the body (`errs.RespondWithValidationError` on failure), call the service, `errs.RespondWithError(c, err)` on failure, map to the response type with one small function per module. `internal/example/handler.go` is the reference.
- Do not log errors in handlers. `RespondWithError` hands every 5xx to the error middleware, which logs it once with the request ID; 4xx are not logged.
- `internal.RespondMessage(c, "...")` for actions with nothing to return; `internal.FormatTime(ts)` for timestamps.

### List caching
Put a per-owner version in the cache key and change it on every write (see `example_service.go`). Never try to enumerate the keys to forget.

## Types.go Pattern

### Rule
- **All request/response types** go in `types.go` within each module
- Swagger marks every field required (`--requiredByDefault`), because gogo-front generates its TypeScript types from it. Tag a field that may be absent with `validate:"optional"`, and an enum with `enums:"a,b,c"`.
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
make seed             # Accounts and sample data for an empty development database
make cli migrate up   # Any console command; flags go after -- (make cli -- create-user --email a@b.c --password secret123)
make cli-create-user EMAIL=a@b.c PASSWORD=secret123   # Create a user (default role super-admin)
make migrate-up       # Apply migrations
make sqlc             # Generate sqlc code
make swagger          # Generate API docs (then run `make api-types` in gogo-front)
make up / make down   # Postgres, Redis and Mailpit in Docker
```

## Code Conventions
- **Handlers**: `GetUser`, `CreateUser`, `ListUsers`
- **Services**: `UserService`, `OrderService`
- **SQL queries**: `GetUserByID`, `CreateUser`, `ListUsers`
- **Files**: `user_service.go`, `order_handler.go`
- **Cache keys**: `user:123`, `examples:user:123:v<version>:page:1:size:20`

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
- **Test Database Guard** - `config.AssertTestDatabaseURL` refuses any database whose name does not end in `_test`, or a non-local host unless `ALLOW_REMOTE_TEST_DB=1`. It also guards `--test-db` and `--test`.
- **One transaction per test** - a failed statement (for example a unique violation) aborts it, so make that request the last one in the test
- **Same wiring as production** - `CreateTestServer` uses `server.NewEngine` and `server.RegisterRoutes`
- **MemoryCache** - Tests use in-memory cache (no Redis required)
- **MemorySender** - Tests collect mail in memory (`server.Mail.Sent()`)
- **GenerateTestJWT** - Helper for authenticated integration requests
- **Factories** - `internal/factory` builds every record a test needs; `helpers.Actor(user)` turns a user into the actor a service expects

### Test Patterns
```go
func TestServiceMethod(t *testing.T) {
    helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
        user := factory.User(t, tx)
        example := factory.Example(t, tx, user.ID)

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
