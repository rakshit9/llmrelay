.PHONY: up down migrate test lint proxy

# Start Postgres + Redis
up:
	docker compose up -d --wait

# Stop everything
down:
	docker compose down

# Run SQL migrations against local Postgres
migrate:
	docker exec -i llmrelay-postgres-1 psql -U llmrelay -d llmrelay < migrations/001_initial.sql

# Build the Go proxy binary
proxy:
	cd proxy && go build -o ../bin/proxy ./cmd/proxy

# Run Go tests
test:
	cd proxy && go test ./...

# Run Go linter
lint:
	cd proxy && go vet ./...
