# Gogo API Tests

Tests run against a real PostgreSQL database. Each test gets its own transaction, rolled back when
the test ends — the same idea as Laravel's `RefreshDatabase`.

## Writing a test

Everything starts from `testkit.New(t)`. There is nothing to close and no context to pass around.

An API test (`tests/integration`):

```go
func TestExampleAPI_CreateExample(t *testing.T) {
    app := testkit.New(t)
    user := app.Create.User()

    app.As(user).POST("/api/v1/examples", example.CreateExampleRequest{Title: "Hello"}).
        AssertOK().
        AssertJSONPath("data.title", "Hello")

    app.POST("/api/v1/examples", example.CreateExampleRequest{Title: "Hello"}).AssertUnauthorized()
    app.As(user).POST("/api/v1/examples", `{}`).AssertInvalid("title")

    app.AssertDatabaseHas("examples", testkit.Row{"user_id": user.ID, "title": "Hello"})
}
```

A service test (`tests/unit`):

```go
func TestExampleService_GetExample(t *testing.T) {
    app := testkit.New(t)
    user := app.Create.User()
    record := app.Create.Example(user.ID)
    service := example.NewExampleService(app.Queries, nil)

    result, err := service.GetExample(app.Ctx, testkit.Actor(user), record.ID)

    require.NoError(t, err)
    assert.Equal(t, record.Title, result.Title)
}
```

## Cheat sheet

| Laravel | Here |
|---|---|
| `use RefreshDatabase` | `app := testkit.New(t)` |
| `User::factory()->create()` | `app.Create.User()` |
| `User::factory()->admin()->create(['email' => $e])` | `app.Create.User(factory.Admin(), factory.WithEmail(e))` |
| `$this->getJson($url)` | `app.GET("/api/v1/examples/%d", id)` |
| `$this->actingAs($user)->postJson($url, $data)` | `app.As(user).POST("/api/v1/examples", data)` |
| `->assertOk()`, `->assertForbidden()` | `.AssertOK()`, `.AssertForbidden()` |
| `->assertJsonPath('data.0.title', 'Hi')` | `.AssertJSONPath("data.0.title", "Hi")` |
| `->assertJsonCount(3, 'data')` | `.AssertJSONCount("data", 3)` |
| `->assertJsonValidationErrors(['title'])` | `.AssertInvalid("title")` |
| `$this->assertDatabaseHas('examples', [...])` | `app.AssertDatabaseHas("examples", testkit.Row{...})` |
| `Mail::fake()` / `Mail::assertSent` | `app.Mail.Sent()` |
| `config(['app.x' => false])` | `testkit.New(t, func(cfg *config.Config) { cfg.X = false })` |

### `app` (`tests/testkit/app.go`)

- `app.Create` — the factories, below
- `app.Ctx`, `app.Queries`, `app.Tx`, `app.TxRunner`, `app.Config`, `app.Logger`, `app.Cache` — what a service constructor asks for
- `app.Mail.Sent()` — every email the application sent during the test
- `app.Token(user)`, `testkit.Actor(user)`, `testkit.JPEG`, `testkit.PDF`

### Requests (`tests/testkit/request.go`)

- `app.GET(path, args...)`, `app.DELETE(path, args...)`
- `app.POST(path, body, args...)`, `PUT`, `PATCH` — the body is a struct, a map or a string of raw JSON; `nil` sends none
- `app.POSTFile(path, field, testkit.File{Name: "a.jpg", Content: testkit.JPEG})`
- `app.As(user)` — the same verbs, logged in as the user
- `app.WithToken(token)`, `app.WithHeader(key, value)`

The path takes `fmt.Sprintf` arguments: `app.PUT("/api/v1/examples/%d", body, id)`.

### Response (`tests/testkit/response.go`)

Every `Assert` stops the test on failure, prints the response body, and returns the response so calls chain.

- `AssertOK()`, `AssertBadRequest()`, `AssertUnauthorized()`, `AssertForbidden()`, `AssertNotFound()`, `AssertTooManyRequests()`, `AssertStatus(code)`
- `AssertError(status, errs.ErrKeyX)` — status and `error_key`
- `AssertInvalid(fields...)` — validation failed, and each field has an error
- `AssertJSONPath(path, want)`, `AssertJSONCount(path, n)`, `AssertJSONMissing(path)`
- `JSON(&v)` decodes the body when you want the typed struct; `resp.StatusCode`, `resp.Body`, `resp.Header` are there too

### Database (`tests/testkit/database.go`)

- `app.AssertDatabaseHas(table, testkit.Row{...})`
- `app.AssertDatabaseMissing(table, testkit.Row{...})`
- `app.AssertDatabaseCount(table, n)` or `app.AssertDatabaseCount(table, n, testkit.Row{...})`

### Factories (`internal/factory`)

| Factory | Options |
|---|---|
| `app.Create.User()` | `WithEmail`, `WithName`, `WithPassword`, `WithRoles`, `Admin()`, `SuperAdmin()`, `Unverified()` |
| `app.Create.Example(userID)` | `WithTitle`, `WithDescription` |
| `app.Create.Examples(userID, count, prefix)` | |
| `app.Create.Upload(userID)` | |

A user has the password `factory.DefaultPassword`. Tests never write `INSERT` by hand, and neither
does a factory: every factory writes through a `Factory*` query in `internal/db/queries/factory.sql`,
the same as application code goes through its own module's queries.

## Adding a factory

A query in `internal/db/queries/factory.sql`, named `Factory<Thing>`, `RETURNING *` so it returns
the generated `db.X` struct directly, then `make sqlc`:

```sql
-- name: FactoryCreateOrder :one
INSERT INTO orders (user_id, status) VALUES (sqlc.arg(user_id), sqlc.arg(status))
RETURNING *;
```

One file in `internal/factory`, shaped like `example.go`:

```go
type orderAttrs struct {
    status string
}

type OrderOption func(*orderAttrs)

func WithStatus(status string) OrderOption { return func(a *orderAttrs) { a.status = status } }

func (f *Factory) Order(userID int32, opts ...OrderOption) *db.Order {
    f.t.Helper()

    attrs := orderAttrs{status: "new"}
    for _, opt := range opts {
        opt(&attrs)
    }

    order, err := f.queries.FactoryCreateOrder(ctx, db.FactoryCreateOrderParams{
        UserID: userID,
        Status: attrs.status,
    })
    if err != nil {
        f.t.Fatalf("factory.Order: %v", err)
    }
    return &order
}
```

## Things to know

- One transaction per test: a failed statement (a unique violation, say) aborts it, so make that request the last one.
- The test server is the production engine and routes (`server.NewEngine`, `server.RegisterRoutes`) and starts on the first request.
- `tests/unit` holds service, policy and pure-logic tests; `tests/integration` holds HTTP tests. Both use the database.
- A helper used by one test file lives at the top of that file (see `newAuthService` in `tests/unit/auth_service_test.go`).

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
