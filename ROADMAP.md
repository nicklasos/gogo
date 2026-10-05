# Roadmap

What is worth adding to the skeleton next, in priority order. Each item names the project it can be ported from. Paths are relative to the directory that holds `gogo`.

## P1

| Item | Why | Port from |
|---|---|---|
| golangci-lint | CI runs `gofmt`, `go vet` and the tests; a linter config would catch more | — |
| Remove `?token=` query auth | Puts JWTs into URLs and access logs; keep only if a project needs WebSockets | `internal/middleware/user_auth.go` |
| Email normalisation | Login matches the email exactly, while the index is on `LOWER(email)`; lower-case on write and on lookup | — |
| Refresh token hashing | Refresh tokens are stored in plain text; store a hash like `auth_tokens` does | `internal/auth/auth_service.go` |

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
| Seeders | `cli seed` for sample data | — |
| Savepoint-per-request in tests | A failed statement aborts the shared test transaction, so such a request must be last in a test | — |

## P3

| Item | Why | Port from |
|---|---|---|
| `make:module` generator | Copy the example module with name substitution, once the module shape is stable | — |
| Audit logs | Who changed what, with a matching modal in the front | `smartcity-backoffice-api/internal/audit_logs/` |
| Client-errors endpoint | Collects front-end errors | `smartcity-backoffice-api/internal/client_errors/` |

## Left out on purpose

- **Queue / background jobs**: cron plus goroutines cover a skeleton; pick River or asynq per project.
- **Keyed lock**: in-process only, misleading once there is more than one instance.
- **Per-module permissions (JSONB)** and **multi-tenancy**: project-specific; roles are enough here.
