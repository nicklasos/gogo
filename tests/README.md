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

2. Set `TEST_DATABASE_URL` in `.env` (or export it). The database name must end in `_test` and the host must be local, otherwise the suite refuses to run (`ALLOW_REMOTE_TEST_DB=1` lifts the host check):
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
- `factory.User`, `factory.Example`, `factory.Upload` (`internal/factory`): test data with defaults and options
- `GenerateTestJWT`: Signed JWT for authenticated requests

## Example Test Pattern

```go
func TestExampleService_GetExample(t *testing.T) {
    helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
        // Setup: Create test data
        user := factory.User(t, tx)
        example := factory.Example(t, tx, user.ID)

        // Test: Execute business logic
        service := example.NewExampleService(queries, nil)
        result, err := service.GetExample(ctx, example.ID, user.ID)

        // Assert: Verify results
        require.NoError(t, err)
        assert.Equal(t, example.Title, result.Title)
    })
}
```

## Mutation Testing

Coverage shows which lines ran; mutation testing shows whether a test would notice if they were wrong.
[Gremlins](https://gremlins.dev) makes one small change at a time (`<` to `<=`, `+` to `-`, `==` to `!=`, `!cond`)
and reruns the tests:

- **KILLED**: a test failed, so the change was caught. Good.
- **LIVED**: every test still passed. A behaviour is not asserted; add or tighten a test for that line.
- **NOT COVERED**: no test runs that line at all.

```bash
make mutation-install                    # once
make mutation-diff                       # code changed since origin/main (BASE=... to change)
make mutation PKG=./internal/auth        # one package
make mutation-dry PKG=./internal/auth    # list the mutants, run nothing
make mutation                            # the whole module
```

The summary ends with **test efficacy** (killed / (killed + lived)) and **mutator coverage**.
`mutation-report.json` has every mutant; list the survivors with:

```bash
jq -r '.files[] | .file_name as $f | .mutations[] | select(.status=="LIVED") | "\($f):\(.line):\(.column) \(.type)"' mutation-report.json
```

Then fix one survivor at a time:

| Type | Change | Usually missing |
|---|---|---|
| `CONDITIONALS_BOUNDARY` | `<` and `<=`, `>` and `>=` | a case exactly at the limit |
| `CONDITIONALS_NEGATION` | `==` and `!=`, `if x` and `if !x` | a test of the other branch |
| `ARITHMETIC_BASE` | `+` and `-`, `*` and `/` | an assertion on the computed value |
| `INCREMENT_DECREMENT` | `i++` and `i--` | an assertion on a count or offset |
| `INVERT_NEGATIVES` | `-x` and `x` | an assertion on the sign |

Some survivors are equivalent (the change cannot alter behaviour); leave those.

**How it runs.** The tests live in `tests/`, not next to the code, so `.gremlins.yaml` sets `integration`
(every mutant runs the whole suite against `TEST_DATABASE_URL`) and `coverpkg: ./internal/...` (coverage
from `tests/` counts for `internal/`). Generated code, factories and `cmd/` are excluded. Each test has its
own rolled-back transaction, so the parallel workers do not interfere.

Gremlins reads your code and measures coverage in the first seconds, then works on copies in a temp folder:
once the first mutants are reported you can keep editing. Do not run two mutation runs at once; they compete
for the CPU and the test database, and slow mutants are reported as TIMED OUT.

Mutation tests are run by hand and are not part of `make test` or CI. A run reports and does not fail;
set `threshold.efficacy` in `.gremlins.yaml` to make it exit non-zero below that score.

## Configuration

The test suite uses environment variables for configuration:
- `TEST_DATABASE_URL`: Connection string for test database

## Benefits

1. **Fast**: Transaction rollback is much faster than recreating data
2. **Isolated**: Each test runs in its own transaction
3. **Real**: Tests actual database behavior and constraints
4. **Safe**: No risk of corrupting development or production data
5. **Maintainable**: Clear separation between unit and integration tests
