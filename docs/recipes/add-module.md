# Add a module

A module is one resource: a table, its queries, and the endpoints over it. The steps below add
`orders`; copy `internal/example` and rename as you go.

## 1. Schema and queries

1. Create `migrations/NNN_create_orders_table.sql` with the next number. Follow [add-migration.md](add-migration.md).
   Include `id SERIAL PRIMARY KEY`, `created_at` and `updated_at` (`TIMESTAMP DEFAULT CURRENT_TIMESTAMP`),
   and `user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE` when rows belong to a user.
2. Create `internal/db/queries/orders.sql`. Typical set: `GetOrderByID :one`, `CreateOrder :one`,
   `UpdateOrder :one`, `DeleteOrder :exec`, `ListOrdersForUserPaginated :many` (`LIMIT $2 OFFSET $3`),
   `CountOrdersForUser :one`. Look a record up by ID alone; who may touch it is the policy's job.
3. `make migrate-up && make sqlc`.

## 2. The module: `internal/orders/`

| File | Contents |
|---|---|
| `types.go` | Request and response structs. `binding:"required"` for required input; `validate:"optional"` for a field that may be absent; `enums:"a,b"` for a fixed set of values. |
| `policy.go` | `CanView`, `CanUpdate`, `CanDelete(actor middleware.Actor, order db.Order) bool`. Pure functions, no database. |
| `order_service.go` | Business logic. Takes `*db.Queries` (and `*db.TxRunner`, `cache.Cache`, `mail.Sender` only if used). Methods take `ctx, actor, ...`, load the record, apply the policy, return domain errors. |
| `handler.go` | One method per route: helpers, bind, call the service, map to the response type. |
| `routes.go` | `RegisterRoutes(app *internal.App)`: builds the service and handler, declares the routes. The only file that sees `*internal.App`. |

Details that are easy to get wrong:

- **Errors**: declare them once at the top of the service (`ErrOrderNotFound = errs.NewNotFoundError(errs.ErrKeyOrderNotFound, "...")`)
  and add the keys to `internal/errs/keys.go` as `orders.<what>`. Map `pgx.ErrNoRows` to the not-found error;
  wrap everything else with `errs.WrapDatabaseError`.
- **Policy denial** is 403 with `errs.ErrKeyForbidden`; a missing record is 404. Load first, then check.
- **Handlers** start with `middleware.CurrentActor(c)`, `middleware.PathID(c, "id")`, `middleware.Page(c)`.
  Each answers the error itself: `if !ok { return }`. Bind with `c.ShouldBindJSON` and answer
  `errs.RespondWithValidationError(c, err)`; answer service errors with `errs.RespondWithError(c, err)`.
- **No logging in handlers.** A 5xx is logged once by the error middleware.
- **Responses**: `{ "data": ... }` for one record, `{ "data": [...], "pagination": ... }` for a list
  (`internal.NewPaginationMeta`), `internal.RespondMessage(c, "...")` when there is nothing to return.
  Timestamps through `internal.FormatTime`.
- **Two writes that must both happen** go inside `s.tx.WithTx(ctx, func(q *db.Queries) error { ... })`.
- **Role-restricted routes**: `group.Use(app.AuthMiddleware, middleware.RequireRole(middleware.RoleAdmin))`.
- **Swagger annotations** on every handler, copied from the example module and adjusted. Errors are `errs.ErrorResponse`.

## 3. Register it

Add `orders.RegisterRoutes(app)` to `internal/server/server.go`. The API and the test server both use that list.

## 4. Factory and seed data

1. `internal/factory/order.go`: `func Order(t TB, conn DB, userID int32, opts ...OrderOption) *db.Order` with defaults for every column.
2. Add a few records to `cmd/cli/internal/commands/seed.go`.

## 5. Tests

- `tests/unit/orders_policy_test.go` (or add to `policy_test.go`): a table of actor × expected result.
- `tests/integration/orders_api_test.go`: for each route the success case, 401 without a token, validation failure,
  404, and one 403 from the policy. Build data with `factory.*`, sign requests with `helpers.GenerateTestJWT`.
- A request that makes a statement fail (a unique violation) must be the last one in its test: it aborts the test's transaction.

## 6. Finish

```bash
make swagger
make test
```

Then, if there is a front end: `make api-types` in gogo-front and follow its `docs/recipes/add-feature.md`.

## Check

- `go build ./... && go vet ./...` are clean, `make test` passes.
- `docs/swagger.yaml` lists the new routes with the right request and response types.
- A plain user gets 403, not 200 or 404, on another user's record.
