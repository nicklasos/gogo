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
	go test ./...

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

.PHONY: run dev build sqlc schema-dump schema-dump-data schema-restore swagger test fmt tidy clean air-install