# M8 — PostgreSQL Persistence Foundation

**Status:** ✅ Completed  
**Date:** 2026-05-09  
**Components:** Go server, Docker, PostgreSQL

## Summary

Replaced the in-memory `Store` with a `DataStore` interface backed by a `PostgresStore` implementation. The in-memory `InMemoryStore` is retained as a fallback when `DATABASE_URL` is not set (useful for development and tests).

## Changes

### Refactored: `InMemoryStore` → `DataStore` interface
- Extracted `DataStore` interface from existing `Store` methods
- Renamed `Store` to `InMemoryStore`
- Added `DataStore` interface with 11 methods: `RegisterUser`, `GetUser`, `ValidateToken`, `UploadKeyBundle`, `GetKeyBundle`, `AvailableOTKCount`, `StoreMessage`, `GetPendingEnvelopes`, `AckMessage`, `GetSentMessages`, `DeleteAllUserData`
- Updated `handler.Server` to accept `service.DataStore` interface
- Refactored `cmd/main.go` to use factory function `initStore()` that selects implementation based on `DATABASE_URL` environment variable

### New: `PostgresStore` implementation
- Full PostgreSQL-backed implementation of the `DataStore` interface
- Uses `pgx/v5` connection pool (`pgxpool.Pool`)
- Auto-migrates schema on startup (embedded migration SQL)
- SQL schema:
  - `users` — username, auth_token, display_name, created_at
  - `key_bundles` — username (FK CASCADE), device_id, curve_key, ed_key, signature, timestamps
  - `one_time_keys` — id (BIGSERIAL), username (FK CASCADE), key_value, consumed, created_at
  - `messages` — id, sender, recipient, ciphertext, message_type, sender_curve_key, status, timestamps
- Indexes on OTK lookup (username + consumed) and message queries (recipient + status, sender)
- Transactions used for: UploadKeyBundle (upsert + OTK replace), GetKeyBundle (OTK consume with FOR UPDATE SKIP LOCKED), GetPendingEnvelopes (SELECT + batch UPDATE), DeleteAllUserData (cascade + message cleanup)

### New: Infrastructure
- `docker-compose.yml` — PostgreSQL 17 Alpine + server (with health check)
- `Dockerfile` — multi-stage Go build for production image
- `Makefile` — targets: `db-up`, `db-down`, `db-migrate`, `run`, `test`, `test-integration`, `clean`
- `migrations/` — SQL migration files (`000001_initial_schema.up.sql` / `.down.sql`)

### New: Integration tests
- 20 integration tests covering all PostgresStore methods
- Tests connect to a real PostgreSQL instance via `DATABASE_URL_TEST` env var
- Auto-skip if `DATABASE_URL_TEST` is not set (safe for CI without PostgreSQL)
- Each test gets a clean database state (tables truncated before test)

## Test Results

### Handler tests (in-memory): 33/33 PASS
```
TestRegisterUser_Validation        PASS  (11 sub-tests)
TestRegisterUser_Duplicate         PASS
TestGetUser_NotFound               PASS
TestGetUser_StripsAuthToken        PASS
TestAuthMiddleware_NoAuth          PASS  (4 sub-tests)
TestAuthMiddleware_ValidToken      PASS
TestUploadKeys_RequiresAuth        PASS
TestUploadKeys_Success             PASS
TestGetKeys_RequiresAuth           PASS
TestGetKeys_ConsumesOTK            PASS
TestGetKeys_NoOTKAvailable         PASS
TestSendMessage_RequiresAuth       PASS
TestSendMessage_BindsSenderToAuth  PASS
TestSendMessage_SenderSpoofing     PASS
TestSendMessage_InvalidRecipient   PASS  (6 sub-tests)
TestGetMessages_RequiresAuth       PASS
TestGetMessages_OnlyOwnMessages    PASS
TestAckMessage_RequiresAuth        PASS
TestAckMessage_WrongUserRejected   PASS
TestAckMessage_RecipientSucceeds   PASS
TestMessageStatusTransitions       PASS
TestRateLimit_Returns429           PASS
TestIsValidUsernameChar            PASS  (9 sub-tests)
TestIsValidBase64                  PASS  (5 sub-tests)
```

### PostgresStore integration tests: 20/20 PASS
```
TestPostgresStore_RegisterUser                PASS
TestPostgresStore_RegisterUser_Duplicate       PASS
TestPostgresStore_GetUser                      PASS
TestPostgresStore_ValidateToken                PASS
TestPostgresStore_UploadKeyBundle              PASS
TestPostgresStore_UploadKeyBundle_UserNotFound PASS
TestPostgresStore_GetKeyBundle                 PASS
TestPostgresStore_GetKeyBundle_NoOTK           PASS
TestPostgresStore_GetKeyBundle_UserNotFound    PASS
TestPostgresStore_AvailableOTKCount            PASS
TestPostgresStore_StoreMessage                 PASS
TestPostgresStore_StoreMessage_RecipientNotFound PASS
TestPostgresStore_GetPendingEnvelopes          PASS
TestPostgresStore_AckMessage                   PASS
TestPostgresStore_AckMessage_WrongUser         PASS
TestPostgresStore_AckMessage_NotFound          PASS
TestPostgresStore_GetSentMessages              PASS
TestPostgresStore_DeleteAllUserData             PASS
TestPostgresStore_DeleteAllUserData_NotFound    PASS
TestPostgresStore_MessageLifecycle              PASS
```

### Go vet: No issues

## API Contract Verification
- All existing handler tests continue to pass with the interface refactor
- PostgresStore returns identical data shapes as InMemoryStore
- Token validation, OTK consumption, message lifecycle all match

## How to Run

```sh
# Start PostgreSQL
cd server && docker compose up -d postgres

# Run all tests (handler + integration)
DATABASE_URL_TEST=postgres://yup:yup_dev@localhost:5432/yup?sslmode=disable go test ./... -count=1

# Start server with PostgreSQL
DATABASE_URL=postgres://yup:yup_dev@localhost:5432/yup?sslmode=disable go run ./cmd/main.go

# Start server with in-memory (no persistence)
go run ./cmd/main.go

# Clean up
docker compose down
```

## Files Created/Modified

| File | Action |
|------|--------|
| `server/internal/service/store.go` | Modified — extracted DataStore, renamed Store → InMemoryStore |
| `server/internal/service/postgres_store.go` | New — PostgresStore implementation |
| `server/internal/service/postgres_store_test.go` | New — 20 integration tests |
| `server/internal/handler/handler.go` | Modified — uses DataStore interface |
| `server/internal/handler/handler_test.go` | Modified — uses NewInMemoryStore |
| `server/cmd/main.go` | Modified — initStore factory, Postgres wiring |
| `server/docker-compose.yml` | New — PostgreSQL + server |
| `server/Dockerfile` | New — multi-stage build |
| `server/Makefile` | New — dev workflow targets |
| `server/migrations/000001_initial_schema.up.sql` | New — schema |
| `server/migrations/000001_initial_schema.down.sql` | New — rollback |
| `server/go.mod` | Modified — added pgx/v5 |
| `server/go.sum` | New — dependency checksums |

## Blockers
- Docker Desktop must be running for PostgreSQL container
- DATABASE_URL_TEST must be set for integration tests
- No new blockers identified
