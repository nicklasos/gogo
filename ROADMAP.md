# Roadmap

What is worth adding to the skeleton next, in priority order. Each item names the project it can be ported from. Paths are relative to the directory that holds `gogo`.

## P1

| Item | Why | Port from |
|---|---|---|
| Background jobs | Mail and other slow work run inside the request today; River (Postgres) fits the stack, with a jobs page in gopulse | — |
| golangci-lint | CI runs `gofmt`, `go vet` and the tests; a linter config would catch more | — |
| WebSocket authentication | Tokens are no longer accepted in the URL; a project with WebSockets needs a short-lived ticket endpoint instead | — |

## P2

| Item | Why | Port from |
|---|---|---|
| S3 / GCS storage | The `Storage` interface is in place with a local implementation only | — |
| Image crop and resize | Useful once a project has avatars or galleries; needs an imaging library | `smartcity-backoffice-api/internal/uploads/crop.go` |
| Role editing | Roles are fixed at creation today; add `UpdateUserRoles` with a last-super-admin guard when a project needs it | — |
| Pagination, sort and filter conventions | Only page parsing exists; every list endpoint reinvents filters | `smartcity-backoffice-api/internal/pagination/` |
| Constraint registry | Table-driven unique / foreign key constraint → error key, instead of string matching | `sytno/backend/internal/errs/postgres_fk.go` |
| pgtype helpers | Less boilerplate converting nullable columns | `sytno/backend/internal/utils/pgtype.go` |
| Mail templates, translations and queueing | The two auth emails are plain English, and `Send` blocks on SMTP | — |
| Savepoint-per-request in tests | A failed statement aborts the shared test transaction, so such a request must be last in a test | — |

## P3

| Item | Why | Port from |
|---|---|---|
| Audit logs | Who changed what, with a matching modal in the front | `smartcity-backoffice-api/internal/audit_logs/` |
| Client-errors endpoint | Collects front-end errors | `smartcity-backoffice-api/internal/client_errors/` |

## Left out on purpose

- **Queue / background jobs**: cron plus goroutines cover a skeleton; pick River or asynq per project.
- **Keyed lock**: in-process only, misleading once there is more than one instance.
- **Per-module permissions (JSONB)** and **multi-tenancy**: project-specific; roles are enough here.
