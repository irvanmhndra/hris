DATABASE_URL ?= postgres://hris:hris_local@127.0.0.1:5440/hris?sslmode=disable
export DATABASE_URL
export SEED_PASSWORD

.PHONY: db migrate migrate-version migrate-baseline api web seed seed-hr lint test build

db:
	docker compose up -d db

# Apply pending migrations (tracked in schema_migrations).
migrate:
	cd hris-api && go run ./cmd/migrate up

migrate-version:
	cd hris-api && go run ./cmd/migrate version

# One-time, for databases created by the old docker-entrypoint-initdb.d mounts:
# records 000001–000004 as applied without re-running them.
migrate-baseline:
	cd hris-api && go run ./cmd/migrate force 4

api:
	cd hris-api && go run ./cmd/api

web:
	cd hris-web && pnpm dev

seed:
	@test -n "$$SEED_PASSWORD" || (echo 'Set SEED_PASSWORD (12–72 characters)'; exit 1)
	cd hris-api && go run ./cmd/seed

seed-hr:
	docker compose exec -T db psql -U hris -d hris -v ON_ERROR_STOP=1 --single-transaction < hris-api/seed/hr_suite.sql

lint:
	cd hris-api && go vet ./... && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...

test:
	cd hris-api && TEST_DATABASE_URL='$(DATABASE_URL)' go test ./...
	cd hris-web && pnpm typecheck

build:
	cd hris-api && go build -o bin/hris-api ./cmd/api
	cd hris-web && pnpm build
