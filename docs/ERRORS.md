# Error Handling Guide

Simple guide for using the `errs` package for consistent error handling.

## Quick Start

### 1. Define Errors in Services

```go
package example

import "app/internal/errs"

var (
    ErrExampleNotFound = errs.NewNotFoundError(errs.ErrKeyExampleNotFound, "Example not found")
)
```

### 2. Return Errors from Services

```go
func (s *Service) GetExample(ctx context.Context, id int32) (*db.Example, error) {
    example, err := s.queries.GetExampleByID(ctx, id)
    if err != nil {
        return nil, ErrExampleNotFound  // Return domain error
    }
    return &example, nil
}
```

### 3. Handle Errors in Handlers

```go
func (h *Handler) GetExample(c *gin.Context) {
    example, err := h.service.GetExample(ctx, id)
    if err != nil {
        errs.RespondWithError(c, err)  // Automatically formats response
        return
    }
    c.JSON(http.StatusOK, response)
}
```

## Error Types

```go
// Not Found (404)
errs.NewNotFoundError(key, message)

// Bad Request (400)
errs.NewBadRequestError(key, message)

// Unauthorized (401)
errs.NewUnauthorizedError(key, message)

// Forbidden (403)
errs.NewForbiddenError(key, message)

// Internal Error (500)
errs.NewInternalError(key, message)
```

## Error Keys

Add error keys in `internal/errs/keys.go`:

```go
const (
    ErrKeyExampleNotFound = "examples.not_found"
)
```

## Handler Helpers

```go
// Validation errors
errs.RespondWithValidationError(c, err)

// Quick responses
errs.RespondWithUnauthorized(c, "Unauthorized")
errs.RespondWithNotFound(c, key, message)
errs.RespondWithBadRequest(c, key, message)
```

## Wrapping Errors

```go
// Wrap database/internal errors
if err != nil {
    return nil, errs.WrapInternal(key, "Failed to create", err)
}
```

## Response Format

All errors return:

```json
{
  "error_key": "examples.not_found",
  "message": "Example not found",
  "status": 404,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## Examples

See `internal/example/example_service.go` and `internal/example/handler.go` for complete examples.
