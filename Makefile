run:
	go run ./cmd/api

dev:
	air

build:
	go build -o bin/api ./cmd/api

sqlc:
	sqlc generate

# Database schema commands
schema-dump:
	@if [ -z "$(DATABASE_URL)" ]; then echo "ERROR: DATABASE_URL not found in .env file or environment"; exit 1; fi
	@echo "Dumping database schema to internal/db/schema.sql..."
	pg_dump --schema-only --no-comments --no-owner --no-privileges "$(DATABASE_URL)" > internal/db/schema.sql
	@echo "Schema dumped successfully!"

swagger:
	swag init --parseDependency --parseInternal -g cmd/api/main.go

test:
	go test ./tests/unit/... ./tests/integration/...

test-unit:
	go test ./tests/unit/...

test-integration:
	go test ./tests/integration/...

test-verbose:
	go test -v ./tests/unit/... ./tests/integration/...

test-coverage:
	go test -cover ./tests/unit/... ./tests/integration/...

# Test database setup
test-db-setup:
	@if [ -z "$(TEST_DATABASE_URL)" ]; then echo "ERROR: TEST_DATABASE_URL not found in .env file or environment"; exit 1; fi
	@echo "Setting up test database..."
	DATABASE_URL="$(TEST_DATABASE_URL)" $(MAKE) migrate-up
	@echo "Test database ready!"

test-db-reset:
	@if [ -z "$(TEST_DATABASE_URL)" ]; then echo "ERROR: TEST_DATABASE_URL not found in .env file or environment"; exit 1; fi
	@echo "Resetting test database..."
	DATABASE_URL="$(TEST_DATABASE_URL)" $(MAKE) migrate-reset
	DATABASE_URL="$(TEST_DATABASE_URL)" $(MAKE) migrate-up
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

# Migration using Go binary (alternative to goose CLI)
migrate-go-up:
	go run cmd/migrate/main.go up

migrate-go-down:
	go run cmd/migrate/main.go down

migrate-go-status:
	go run cmd/migrate/main.go status

migrate-go-create:
	@read -p "Enter migration name: " name; \
	go run cmd/migrate/main.go create $$name

tidy:
	go mod tidy

clean:
	rm -rf bin/ docs/ tmp/

air-install:
	go install github.com/cosmtrek/air@latest

# Supervisord commands
supervisor-restart:
	sudo supervisorctl restart smartcity-api

supervisor-status:
	sudo supervisorctl status smartcity-api

supervisor-stop:
	sudo supervisorctl stop smartcity-api

supervisor-start:
	sudo supervisorctl start smartcity-api

supervisor-logs:
	sudo tail -f /var/log/smartcity/smartcity-api.log

supervisor-error-logs:
	sudo tail -f /var/log/smartcity/smartcity-api-error.log

.PHONY: run dev build sqlc schema-dump schema-dump-data schema-restore swagger test test-unit test-integration test-verbose test-coverage test-db-setup test-db-reset test-with-db test-migrate-up test-migrate-down test-migrate-status fmt tidy clean air-install supervisor-restart supervisor-status supervisor-stop supervisor-start supervisor-logs supervisor-error-logs