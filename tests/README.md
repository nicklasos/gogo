# Gogo API Tests

This directory contains the test suite for the Gogo API template using the Laravel-style transaction rollback pattern.

## Test Structure

```
tests/
├── helpers/          # Test utilities and helpers
│   ├── db_helper.go      # Database transaction helpers
│   ├── test_server.go    # HTTP test server setup
│   └── fixtures.go       # Test data fixtures
├── unit/             # Unit tests (services, pure logic)
│   ├── auth_service_test.go
│   ├── example_service_test.go
│   └── upload_service_test.go
├── integration/      # API integration tests
│   ├── auth_api_test.go
│   ├── example_api_test.go
│   └── upload_api_test.go
├── test_config.go    # Test configuration
└── README.md         # This file
```

## Running Tests

### Prerequisites
1. Set up a test database (separate from development):
   ```bash
   createdb gogo_test
   ```

2. Set `TEST_DATABASE_URL` in `.env` (or export it):
   ```bash
   export TEST_DATABASE_URL="postgres://postgres@localhost:5432/gogo_test?sslmode=disable"
   ```

3. Run tests (migrations are applied automatically via Make):
   ```bash
   make test
   ```

### Running Tests

```bash
# Run all tests (auto-migrates test DB when TEST_DATABASE_URL is set)
make test

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Run with verbose output
make test-verbose

# Run specific test
go test -v ./tests/unit -run TestAuthService_Register
```

## Test Features

### Laravel-Style Transaction Rollback
- Each test runs in its own database transaction
- Automatic rollback after test completion
- No data pollution between tests
- Fast execution (transactions are faster than recreation)

### Real Database Testing
- Uses actual PostgreSQL database
- Tests real SQL queries and constraints
- No mocking of database layer
- Type-safe with sqlc integration

### Two Test Types
1. **Unit Tests**: Service layer business logic
2. **Integration Tests**: Full HTTP API endpoints

### Test Helpers
- `WithTransaction`: Database transaction wrapper
- `CreateTestServer`: HTTP test server setup
- `CreateTestUser`, `CreateTestExample`: Test data fixtures
- `GenerateTestJWT`: Signed JWT for authenticated requests

## Example Test Pattern

```go
func TestExampleService_GetExample(t *testing.T) {
    helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
        // Setup: Create test data
        user := helpers.CreateTestUser(t, ctx, tx)
        example := helpers.CreateTestExample(t, ctx, tx, user.ID)

        // Test: Execute business logic
        service := example.NewExampleService(queries, nil)
        result, err := service.GetExample(ctx, example.ID, user.ID)

        // Assert: Verify results
        require.NoError(t, err)
        assert.Equal(t, example.Title, result.Title)
    })
}
```

## Configuration

The test suite uses environment variables for configuration:
- `TEST_DATABASE_URL`: Connection string for test database

## Benefits

1. **Fast**: Transaction rollback is much faster than recreating data
2. **Isolated**: Each test runs in its own transaction
3. **Real**: Tests actual database behavior and constraints
4. **Safe**: No risk of corrupting development or production data
5. **Maintainable**: Clear separation between unit and integration tests
