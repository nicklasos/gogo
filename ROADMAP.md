# Roadmap

What is worth adding to the skeleton next, in priority order. Each item names the project it can be ported from. Paths are relative to the directory that holds `gogo`.

## P1

| Item | Why | Port from |
|---|---|---|
| Auth rate limiter + `cache.Increment` | `/auth/login` and `/auth/refresh` can be brute-forced | `smartcity-backoffice-api/internal/middleware/rate_limit.go`, `internal/cache/cache.go` |
| Config hardening | `getEnvInt` / `getEnvDuration`, refuse a weak `JWT_SECRET`, env-driven token TTLs (access is hardcoded to 7 days), `SetTrustedProxies` | `smartcity-backoffice-api/config/config.go`, `cmd/api/main.go` |
| TypeScript types generated from OpenAPI | Removes hand-written types in gogo-front; swagger is correct now, so it is unblocked | — |
| CI + golangci-lint | No pipeline yet; needs Postgres and Redis services, goose, `make test` | — |
| Dockerfile + compose (Postgres, Redis) | One-command local setup and a deployable image | `sytno/backend/Dockerfile`, `docker-entrypoint.sh` |
| `ALLOW_REGISTRATION` flag | `/auth/register` is open to anyone | — |
| Remove `?token=` query auth | Puts JWTs into URLs and access logs; keep only if a project needs WebSockets | `internal/middleware/user_auth.go` |

## P2

| Item | Why | Port from |
|---|---|---|
| Uploads hardening | Path traversal guard, content sniffing instead of trusting the client MIME type, no directory listing, storage interface (local / S3), optional crop | `smartcity-backoffice-api/internal/uploads/` |
| Role editing | Roles are fixed at creation today; add `UpdateUserRoles` with a last-super-admin guard when a project needs it | — |
| Pagination, sort and filter conventions | Only page parsing exists; every list endpoint reinvents filters | `smartcity-backoffice-api/internal/pagination/` |
| Constraint registry | Table-driven unique / foreign key constraint → error key, instead of string matching | `sytno/backend/internal/errs/postgres_fk.go` |
| pgtype helpers | Less boilerplate converting nullable columns | `sytno/backend/internal/utils/pgtype.go` |
| Mail + password reset | Needs an SMTP service and a reset-token table | `smartcity-api/internal/mail/mail.go` |
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
