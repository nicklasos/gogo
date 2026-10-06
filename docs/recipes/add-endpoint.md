# Add an endpoint to an existing module

1. **Query** (if needed): add it to `internal/db/queries/<module>.sql`, then `make sqlc`.
2. **Types**: add the request and response structs to the module's `types.go`.
   Reuse the module's existing response type when the shape is the same.
3. **Policy**: if the action has its own rule, add a `CanX(actor, record) bool` to `policy.go`. Otherwise reuse one.
4. **Service**: add the method. Signature `(ctx context.Context, actor middleware.Actor, ...)`.
   Load the record, apply the policy, do the work, return a domain error on failure.
5. **Handler**: add the method with its swagger block. Start with the helpers
   (`middleware.CurrentActor`, `middleware.PathID`, `middleware.Page`), bind, call the service, respond.
6. **Route**: add it in `routes.go`. Routes with a literal segment (`/orders/export`) go before `/:id` routes.
7. **Error keys**: add any new one to `internal/errs/keys.go` and tell the front end, which needs a translation for it.
8. **Tests**: success, 401, validation failure, and the policy's 403 in `tests/integration/<module>_api_test.go`.
9. `make swagger && make test`. If the front end uses it, `make api-types` there.

## Choosing the status code

| Situation | Status | Why |
|---|---|---|
| Body fails validation | 400, `validation.failed` with `errors` per field | The front end puts these under form fields |
| Not signed in, or the token is no longer valid | 401 | Clients react to 401 by refreshing the session, so use it for nothing else |
| Signed in but not allowed | 403 | |
| The record does not exist | 404 | |
| A business rule says no (wrong current password, cannot delete yourself) | 400 with its own key | |
