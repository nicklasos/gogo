run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

sqlc:
	sqlc generate

swagger:
	swag init --parseDependency --parseInternal -g cmd/api/main.go

test:
	go test ./...

fmt:
	go fmt ./...

# Migration commands (requires DATABASE_URL environment variable)
migrate-up:
	@if [ -z "$(DATABASE_URL)" ]; then echo "ERROR: DATABASE_URL environment variable is required"; exit 1; fi
	goose -dir migrations mysql "$(DATABASE_URL)" up

migrate-down:
	@if [ -z "$(DATABASE_URL)" ]; then echo "ERROR: DATABASE_URL environment variable is required"; exit 1; fi
	goose -dir migrations mysql "$(DATABASE_URL)" down

migrate-status:
	@if [ -z "$(DATABASE_URL)" ]; then echo "ERROR: DATABASE_URL environment variable is required"; exit 1; fi
	goose -dir migrations mysql "$(DATABASE_URL)" status

migrate-reset:
	@if [ -z "$(DATABASE_URL)" ]; then echo "ERROR: DATABASE_URL environment variable is required"; exit 1; fi
	goose -dir migrations mysql "$(DATABASE_URL)" reset

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
	rm -rf bin/ docs/

.PHONY: run build sqlc swagger test fmt tidy clean