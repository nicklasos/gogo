# Change the schema or a query

## A new migration

1. Create `migrations/NNN_description.sql`, where `NNN` is the highest existing number plus one.
   Never edit a migration that has been committed: other databases have already run it.
2. Use this shape:

   ```sql
   -- +goose Up
   -- +goose StatementBegin
   ALTER TABLE orders ADD COLUMN note TEXT;
   -- +goose StatementEnd

   -- +goose Down
   -- +goose StatementBegin
   ALTER TABLE orders DROP COLUMN IF EXISTS note;
   -- +goose StatementEnd
   ```

3. A column added to a table with rows needs a default or a backfill in the same migration before `SET NOT NULL`.
4. Apply and regenerate:

   ```bash
   make migrate-up   # development database
   make sqlc         # sqlc reads the schema from migrations/
   ```

   `make test` applies migrations to the test database by itself.

## A new or changed query

1. Edit `internal/db/queries/<module>.sql`. One `-- name: VerbNoun :one|:many|:exec` per query.
   Name parameters with `sqlc.arg(name)` when there are more than two or their meaning is not obvious.
2. `make sqlc`. Never edit `internal/db/*.sql.go` by hand.
3. Fix the callers the compiler points at.

## After either

- A changed column usually means a changed factory (`internal/factory/`) and response type (`types.go`).
- `make test`. If a response type changed: `make swagger`, then `make api-types` in gogo-front.

## Check

- `go build ./...` is clean and `make test` passes.
- `make cli -- --test migrate status` shows the new migration applied.
- The `Down` section really undoes the `Up` section.
