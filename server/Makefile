.PHONY: db-up db-down db-migrate run test test-integration clean

# Start PostgreSQL container
db-up:
	docker compose up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 3
	docker compose exec -T postgres pg_isready -U yup

# Stop PostgreSQL container
db-down:
	docker compose down

# Run database migrations manually (uses golang-migrate CLI)
# Requires db-up first. Credentials must match docker-compose.yml.
db-migrate:
ifeq ($(strip $(CMD)),)
	$(error Usage: make db-migrate CMD="up"  (or "down", "drop", "goto N"))
endif
	docker compose run --rm \
		-v "$(PWD)/migrations:/migrations" \
		--entrypoint sh \
		migrate/migrate:v4.18.1 \
		-c "migrate -path=/migrations -database 'postgres://yup:yup_dev@postgres:5432/yup?sslmode=disable' $(CMD)"

# Run the server locally (requires DATABASE_URL or falls back to in-memory)
run:
	go run ./cmd/main.go

# Run all unit tests (use in-memory store)
test:
	go test ./... -count=1 -v

# Run integration tests against a real PostgreSQL (requires: db-up, DATABASE_URL_TEST=postgres://...)
# Example: make test-integration DATABASE_URL_TEST="postgres://yup:yup_dev@localhost:5432/yup?sslmode=disable"
# Note: Use -p 1 to avoid parallel test interference with shared database state.
test-integration:
	go test ./... -count=1 -v -p 1 -run "TestStoreSuite_Postgres|TestPostgresStore"

# Clean build artifacts
clean:
	go clean ./...
	rm -f main.exe yup-server.exe
