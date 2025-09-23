# SmartCity API Tests

This directory contains the test suite for the SmartCity API using the Laravel-style transaction rollback pattern.

## Test Structure

```
tests/
├── helpers/          # Test utilities and helpers
│   ├── db_helper.go      # Database transaction helpers
│   ├── test_server.go    # HTTP test server setup
│   └── fixtures.go       # Test data fixtures
├── unit/             # Unit tests (services, pure logic)
│   └── news_service_test.go
├── integration/      # API integration tests
│   └── news_api_test.go
├── test_config.go    # Test configuration
└── README.md         # This file
```

## Running Tests

### Prerequisites
1. Set up a test database (separate from development):
   ```bash
   createdb smartcity_test
   ```

2. Run migrations on test database:
   ```bash
   export DATABASE_URL="postgres://postgres:postgres@localhost:5432/smartcity_test?sslmode=disable"
   make migrate-up
   ```

3. Set test database URL:
   ```bash
   export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/smartcity_test?sslmode=disable"
   ```

### Running Tests

```bash
# Run all tests
go test ./tests/...

# Run only unit tests
go test ./tests/unit/...

# Run only integration tests
go test ./tests/integration/...

# Run with verbose output
go test -v ./tests/...

# Run specific test
go test -v ./tests/unit -run TestNewsService_GetNewsByID
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
- `WithTestServer`: HTTP test server setup
- `CreateTestNews`, `CreateTestCity`: Test data fixtures

## Example Test Pattern

```go
func TestNewsService_GetNewsByID(t *testing.T) {
    helpers.WithTransactionQueries(t, func(ctx context.Context, queries *db.Queries) {
        // Setup: Create test data
        city := helpers.CreateTestCity(t, ctx, queries)
        testNews := helpers.CreateTestNews(t, ctx, queries, city.ID)
        
        // Test: Execute business logic
        service := news.NewNewsService(queries)
        result, err := service.GetNewsByID(ctx, testNews.ID)
        
        // Assert: Verify results
        assert.NoError(t, err)
        assert.Equal(t, testNews.Title, result.Title)
    })
}
```

## Configuration

The test suite uses environment variables for configuration:
- `TEST_DATABASE_URL`: Connection string for test database
- Falls back to default: `postgres://postgres:postgres@localhost:5432/smartcity_test?sslmode=disable`

## Benefits

1. **Fast**: Transaction rollback is much faster than recreating data
2. **Isolated**: Each test runs in its own transaction
3. **Real**: Tests actual database behavior and constraints
4. **Safe**: No risk of corrupting development or production data
5. **Maintainable**: Clear separation between unit and integration tests