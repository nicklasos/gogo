# Add a scheduled job

Jobs run on a cron schedule, inside the API process when `ENABLE_SCHEDULER=true` or in the
separate `cmd/cron` binary. Run them in one place only, or every job runs twice.

1. **The job**: `internal/scheduler/jobs/<name>.go`, modelled on `cleanup_refresh_tokens.go`.
   A struct holding what it needs (`*db.Queries`, the logger, config) with `Execute(ctx context.Context) error`.
   Put the real work in a service method when a module already owns that logic, and call it from the job.
2. **Register it**: add a `register<Name>Job()` method to `internal/scheduler/scheduler.go` and call it from `RegisterJobs`.
   The schedule is a cron expression or a shortcut such as `@daily`, `@hourly`, `@every 5m`.
3. **Make it safe to repeat.** A job can be interrupted by a deploy and run again later:
   delete or update by condition (`WHERE expires_at < now()`), never by a list computed earlier.
4. **Log one line at the start and one at the end**, with counts when there are any. Return the error; the scheduler logs it.
5. **Test** `Execute` directly in `tests/unit/`, inside `helpers.WithTransaction`, with data from `factory.*`.
6. `make test`.

A job that needs a new dependency (mail, cache) gets it through `scheduler.Dependencies`,
which is filled in `cmd/api/main.go` and `cmd/cron/main.go`: update both.
