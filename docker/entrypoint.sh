#!/bin/sh
set -e

# Migrations run only in front of the API, so a cron or cli container started from the
# same image does not race it. Set RUN_MIGRATIONS=false to run them as a separate step.
if [ "$1" = "/app/api" ] && [ "${RUN_MIGRATIONS:-true}" = "true" ] && [ -n "$DATABASE_URL" ]; then
  echo "Running migrations..."
  /app/goose -dir /app/migrations postgres "$DATABASE_URL" up
fi

exec "$@"
