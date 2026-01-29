# GOGO – Go API Template with SQLC

A production-ready Go API template built with **Gin**, **PostgreSQL**, **Redis**, **SQLC**, and **Goose** migrations.

![Logo](logo.jpg)

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

- **Authentication**: JWT-based auth with registration, login, refresh tokens
- **CRUD Example**: Complete example module demonstrating patterns
- **Pagination**: Context-based pagination middleware
- **Type Safety**: SQLC for type-safe database operations
- **Swagger**: Auto-generated API documentation

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
├── cmd/api/main.go              # Application entry point
├── internal/
│   ├── app.go                   # App context with DB, Cache, Logger
│   ├── auth/                    # Authentication module
│   │   ├── auth_service.go
│   │   ├── handlers.go
│   │   ├── routes.go
│   │   └── types.go
│   ├── example/                 # Example CRUD module
│   │   ├── example_service.go
│   │   ├── handler.go
│   │   ├── routes.go
│   │   └── types.go
│   ├── db/
│   │   ├── queries/              # SQL queries
│   │   └── *.sql.go             # SQLC generated code
│   ├── middleware/
│   │   ├── user_auth.go         # JWT authentication
│   │   └── pagination.go        # Pagination context
│   └── responses.go             # PaginationMeta helper
├── migrations/                  # Goose database migrations
└── sqlc.yaml                    # SQLC configuration
```

## Development Commands

```bash
# Development
make dev              # Hot reload server
make test             # Run all tests

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
- `POST /api/v1/auth/logout` - Logout (protected)

### Examples
- `GET /api/v1/examples` - List examples with pagination (protected)
- `POST /api/v1/examples` - Create example (protected)
- `GET /api/v1/examples/:id` - Get example (protected)
- `PUT /api/v1/examples/:id` - Update example (protected)
- `DELETE /api/v1/examples/:id` - Delete example (protected)

### Uploads
- `POST /api/v1/uploads` - Upload a file (protected)
  - Accepts: `multipart/form-data` with `file` field
  - Returns: Upload ID, relative path, full URL, type, and metadata
  - Supported types: images (jpg, jpeg, png, gif, webp), videos (mp4, avi, mov), documents (pdf, doc, docx, txt), audio (mp3, wav, ogg)
  - Max file size: 50MB (configurable)

### Other
- `GET /health` - Health check
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
```

## Patterns

### Context Pattern
- **User ID**: Use `middleware.GetUserIDFromContext(c)` in handlers
- **Pagination**: Use `middleware.GetPaginationParamsFromContext(c, default, min, max)`

### Types.go Pattern
- All request/response types go in `types.go` within each module
- Services define internal types (e.g., `PaginatedExamplesResult`) in service files
- Handlers convert service types to response types from `types.go`

## Uploads Module

The uploads module allows users to upload files (images, videos, documents, audio) and stores metadata in the database.

### Usage

#### Uploading Files

```go
// In your handler or service
import "app/internal/uploads"

// Get upload service from app context
config := uploads.DefaultUploadConfig(app.Config.UploadFolder, app.Config.FilesBaseURL)
service := uploads.NewUploadService(app.Queries, config)

// Upload file
upload, err := service.UploadFile(ctx, fileHeader, userID)
if err != nil {
    // Handle error
}
// upload.ID, upload.RelativePath, upload.Type, etc.
```

#### Retrieving Uploads

```go
// Get a single upload
upload, err := service.GetUpload(ctx, uploadID, userID)
if err != nil {
    // Handle error (returns ErrUploadNotFound if not found)
}

// List all uploads for a user
uploads, err := service.ListUploads(ctx, userID)
if err != nil {
    // Handle error
}
```

#### Deleting Uploads

```go
// Delete an upload (removes from DB and disk)
err := service.DeleteUpload(ctx, uploadID, userID)
if err != nil {
    // Handle error (returns ErrUploadNotFound if not found)
}
```

#### Configuration

The upload service is configurable:

```go
config := &uploads.UploadConfig{
    UploadFolder: "./uploads",
    BaseURL:      "http://localhost:8181/api/files",
    MaxFileSize:  50 * 1024 * 1024, // 50MB
    AllowedTypes: []string{".jpg", ".png", ".pdf"}, // Custom allowed types
    GetFolderID: func(ctx context.Context, userID int32) (int32, error) {
        // Custom logic to determine folder ID
        // Default: returns userID
        return userID, nil
    },
}
```

### File Types

The service automatically detects file types:
- **image**: jpg, jpeg, png, gif, webp
- **video**: mp4, avi, mov, wmv, flv
- **audio**: mp3, wav, ogg, aac, flac
- **document**: pdf, doc, docx, txt, xls, xlsx
- **other**: any other extension

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
- See `internal/example/` for complete examples

## Links

- **API Docs**: `/swagger/index.html` when running
- **Error Handling**: See `docs/ERRORS.md` for error handling guide
- **Architecture**: See `CLAUDE.md` for detailed patterns
