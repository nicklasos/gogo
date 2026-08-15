run:
	go run ./cmd/api

dev:
	air

build:
	go build -o bin/api ./cmd/api

build-cron:
	go build -o bin/gogo-cron ./cmd/cron

build-cli:
	go build -o bin/gogo-cli ./cmd/cli

run-cron:
	go run ./cmd/cron

run-cron-test-db:
	go run ./cmd/cron --test-db

run-test-db:
	go run ./cmd/api --test-db

sqlc:
	sqlc generate

sqlc-install:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Database schema commands
schema-dump:
	@if [ -z "$(DATABASE_URL)" ]; then echo "ERROR: DATABASE_URL not found in .env file or environment"; exit 1; fi
	@echo "Dumping database schema to internal/db/schema.sql..."
	pg_dump --schema-only --no-comments --no-owner --no-privileges "$(DATABASE_URL)" > internal/db/schema.sql
	@echo "Schema dumped successfully!"

swagger:
	swag init --parseDependency --parseInternal -g cmd/api/main.go

# Apply migrations to TEST_DATABASE_URL before tests so schema matches migrations/
test:
	@if [ -n "$(TEST_DATABASE_URL)" ]; then goose -dir migrations postgres "$(TEST_DATABASE_URL)" up; fi
	go test ./tests/unit/... ./tests/integration/...

test-unit:
	@if [ -n "$(TEST_DATABASE_URL)" ]; then goose -dir migrations postgres "$(TEST_DATABASE_URL)" up; fi
	go test ./tests/unit/...

test-integration:
	@if [ -n "$(TEST_DATABASE_URL)" ]; then goose -dir migrations postgres "$(TEST_DATABASE_URL)" up; fi
	go test ./tests/integration/...

test-verbose:
	@if [ -n "$(TEST_DATABASE_URL)" ]; then goose -dir migrations postgres "$(TEST_DATABASE_URL)" up; fi
	go test -v ./tests/unit/... ./tests/integration/...

test-coverage:
	@if [ -n "$(TEST_DATABASE_URL)" ]; then goose -dir migrations postgres "$(TEST_DATABASE_URL)" up; fi
	go test -cover ./tests/unit/... ./tests/integration/...

# Test database setup
test-db-setup:
	@if [ -z "$(TEST_DATABASE_URL)" ]; then echo "ERROR: TEST_DATABASE_URL not found in .env file or environment"; exit 1; fi
	@echo "Setting up test database..."
	goose -dir migrations postgres "$(TEST_DATABASE_URL)" up
	@echo "Test database ready!"

test-db-reset:
	@if [ -z "$(TEST_DATABASE_URL)" ]; then echo "ERROR: TEST_DATABASE_URL not found in .env file or environment"; exit 1; fi
	@echo "Resetting test database..."
	goose -dir migrations postgres "$(TEST_DATABASE_URL)" reset
	goose -dir migrations postgres "$(TEST_DATABASE_URL)" up
	@echo "Test database reset complete!"

# Run tests with test database setup
test-with-db:
	@if [ -z "$(TEST_DATABASE_URL)" ]; then echo "ERROR: TEST_DATABASE_URL not found in .env file or environment"; exit 1; fi
	@echo "Running tests with test database..."
	$(MAKE) test-db-setup
	go test ./tests/unit/... ./tests/integration/...

# Test database migrations
test-migrate-up:
	@if [ -z "$(TEST_DATABASE_URL)" ]; then echo "ERROR: TEST_DATABASE_URL not found in .env file or environment"; exit 1; fi
	goose -dir migrations postgres "$(TEST_DATABASE_URL)" up

test-migrate-down:
	@if [ -z "$(TEST_DATABASE_URL)" ]; then echo "ERROR: TEST_DATABASE_URL not found in .env file or environment"; exit 1; fi
	goose -dir migrations postgres "$(TEST_DATABASE_URL)" down

test-migrate-status:
	@if [ -z "$(TEST_DATABASE_URL)" ]; then echo "ERROR: TEST_DATABASE_URL not found in .env file or environment"; exit 1; fi
	goose -dir migrations postgres "$(TEST_DATABASE_URL)" status

fmt:
	go fmt ./...

# Load .env file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Migration commands (uses DATABASE_URL from .env or environment)
migrate-up:
	@if [ -z "$(DATABASE_URL)" ]; then echo "ERROR: DATABASE_URL not found in .env file or environment"; exit 1; fi
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	@if [ -z "$(DATABASE_URL)" ]; then echo "ERROR: DATABASE_URL not found in .env file or environment"; exit 1; fi
	goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status:
	@if [ -z "$(DATABASE_URL)" ]; then echo "ERROR: DATABASE_URL not found in .env file or environment"; exit 1; fi
	goose -dir migrations postgres "$(DATABASE_URL)" status

migrate-reset:
	@if [ -z "$(DATABASE_URL)" ]; then echo "ERROR: DATABASE_URL not found in .env file or environment"; exit 1; fi
	goose -dir migrations postgres "$(DATABASE_URL)" reset

migrate-create:
	@read -p "Enter migration name: " name; \
	goose -dir migrations create $$name sql

migrate-install:
	go install github.com/pressly/goose/v3/cmd/goose@latest

# CLI-based migration commands
cli-migrate-up:
	go run ./cmd/cli migrate up

cli-migrate-down:
	go run ./cmd/cli migrate down

cli-migrate-status:
	go run ./cmd/cli migrate status

cli-migrate-create:
	@read -p "Enter migration name: " name; \
	go run ./cmd/cli migrate create $$name

cli-migrate-test-up:
	go run ./cmd/cli migrate --test up

cli-migrate-test-down:
	go run ./cmd/cli migrate --test down

cli-migrate-test-status:
	go run ./cmd/cli migrate --test status

# CLI test command
cli-test:
	go run ./cmd/cli test

cli-test-db:
	go run ./cmd/cli --test test

# CLI help
cli-help:
	go run ./cmd/cli help

tidy:
	go mod tidy

clean:
	rm -rf bin/ docs/ tmp/

air-install:
	go install github.com/cosmtrek/air@latest

# Supervisord commands
supervisor-restart:
	sudo supervisorctl restart gogo-api

supervisor-status:
	sudo supervisorctl status gogo-api

supervisor-stop:
	sudo supervisorctl stop gogo-api

supervisor-start:
	sudo supervisorctl start gogo-api

supervisor-logs:
	sudo tail -f /var/log/app/app.log

supervisor-error-logs:
	sudo tail -f /var/log/app/app-error.log

help:
	@echo "Gogo API Template - available targets:"
	@echo ""
	@echo "  Development:"
	@echo "    run                 Start API server"
	@echo "    run-test-db         Start API with TEST_DATABASE_URL"
	@echo "    dev                 Hot reload (air)"
	@echo "    build               Build API binary"
	@echo "    build-cron          Build cron binary"
	@echo "    build-cli           Build CLI binary"
	@echo "    fmt                 Format Go code"
	@echo "    tidy                go mod tidy"
	@echo ""
	@echo "  Database:"
	@echo "    migrate-up          Apply migrations"
	@echo "    migrate-down        Roll back one migration"
	@echo "    migrate-status      Show migration status"
	@echo "    migrate-reset       Reset all migrations"
	@echo "    migrate-create      Create a new migration"
	@echo "    migrate-install     Install goose CLI"
	@echo "    sqlc                Generate sqlc code"
	@echo "    sqlc-install        Install sqlc CLI"
	@echo "    schema-dump         Dump schema to internal/db/schema.sql"
	@echo ""
	@echo "  Testing:"
	@echo "    test                Run all tests (auto-migrates test DB)"
	@echo "    test-unit           Run unit tests"
	@echo "    test-integration    Run integration tests"
	@echo "    test-verbose        Run tests with -v"
	@echo "    test-coverage       Run tests with coverage"
	@echo "    test-db-setup       Apply migrations to test DB"
	@echo "    test-db-reset       Reset and re-migrate test DB"
	@echo ""
	@echo "  Docs / Cron / Deploy:"
	@echo "    swagger             Generate Swagger docs"
	@echo "    run-cron            Run standalone cron"
	@echo "    supervisor-restart  Restart supervisord program"
	@echo "    help                Show this help"

.PHONY: run dev build build-cron build-cli run-cron run-cron-test-db run-test-db sqlc sqlc-install schema-dump swagger test test-unit test-integration test-verbose test-coverage test-db-setup test-db-reset test-with-db test-migrate-up test-migrate-down test-migrate-status fmt tidy clean air-install migrate-up migrate-down migrate-status migrate-reset migrate-create migrate-install cli-migrate-up cli-migrate-down cli-migrate-status cli-migrate-create cli-migrate-test-up cli-migrate-test-down cli-migrate-test-status cli-test cli-test-db cli-help supervisor-restart supervisor-status supervisor-stop supervisor-start supervisor-logs supervisor-error-logs help
