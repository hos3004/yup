# M8 — PostgreSQL Persistence Report

## Summary

PostgreSQL persistence implemented as `PostgresStore` implementing the `DataStore` interface alongside `InMemoryStore`. Both stores share identical semantics verified through a shared test suite (`handler_store_test.go`). PostgreSQL runs locally via Docker (`postgres:17-alpine`).

## DataStore Interface

All persistence methods defined in `server/internal/service/store.go:13`:

| Method | Description |
|--------|-------------|
| `RegisterUser` | Create user, return raw auth token |
| `GetUser` | Lookup user by username (no token returned) |
| `ValidateToken` | Hash+constant-time token validation |
| `UploadKeyBundle` | Upsert key bundle, replace OTKs |
| `GetKeyBundle` | Fetch bundle, consume one OTK (by row id) |
| `GetCurveKey` | Read registered curve_key for sender binding |
| `AvailableOTKCount` | Count unconsumed OTKs |
| `StoreMessage` | Store pending message, sender derived from auth |
| `GetPendingEnvelopes` | Return pending→delivered messages (once per session) |
| `AckMessage` | Transition pending/delivered→received |
| `GetSentMessages` | Return sent messages (all statuses) |
| `DeleteAllUserData` | Cascade delete user + all related data |

## Schema (PostgreSQL)

See `migrations/000001_initial_schema.up.sql` and `000002_semantics_hardening.up.sql`.

### users
- `username VARCHAR(64) PK`
- `auth_token VARCHAR(128) NOT NULL UNIQUE` (raw token stored for backward compat)
- `token_hash VARCHAR(64) NOT NULL DEFAULT ''` (SHA-256 of token)
- `display_name`, `created_at`

### key_bundles
- `username PK REFERENCES users(username) ON DELETE CASCADE`
- `device_id`, `curve_key`, `ed_key`, `signature`, `created_at`, `updated_at`

### one_time_keys
- `id BIGSERIAL PK`
- `username REFERENCES users(username) ON DELETE CASCADE`
- `key_value VARCHAR(256) NOT NULL`
- `consumed BOOLEAN DEFAULT FALSE`
- `consumed_at TIMESTAMPTZ`
- `UNIQUE(username, key_value)` (via unique index `idx_otk_unique_username_key`)
- Index: `idx_otk_username_consumed ON (username, consumed, consumed_at)`

### messages
- `id VARCHAR(64) PK`
- `sender_username FK → users(username) ON DELETE CASCADE`
- `recipient_username FK → users(username) ON DELETE CASCADE`
- `ciphertext`, `message_type`, `sender_curve_key`
- `status` (pending→delivered→received)
- `created_at`, `delivered_at`

## Delivery State Machine

```
StoreMessage → pending
GetPendingEnvelopes → pending→delivered (returned once per session)
AckMessage → delivered/pending→received
Server restart → delivered→pending (retry after restart)
```

## Token Security

- Tokens generated as 32-byte random hex (64 chars)
- `RegisterUser`: stores both raw `auth_token` and `sha256(auth_token)` as `token_hash`
- `ValidateToken`: hashes incoming token, queries by `token_hash`
- Raw token stored for backward compat; `GetUser` strips it from response

## Startup Behavior

`NewPostgresStore`:
1. Connect with `pgx/v5` pool
2. Run embedded migration SQL (CREATE TABLE IF NOT EXISTS + ALTER TABLE for existing databases)
3. Reset `delivered`→`pending` messages (retry after restart)

## Test Coverage

- **32 handler tests** (InMemory): registration, auth, key upload/fetch, OTK consumption, send/get/ack messages, lifecycle, rate limiting, validation
- **6 shared store suite tests** run against both InMemory and PostgresStore: register+get, token validation, key bundle+OTK, message lifecycle, sent messages
- **20 PG integration tests** (`TestPostgresStore_*`): full coverage of all DataStore methods against real PostgreSQL
- **`go vet ./...`**: clean

## Known Limitations

- Migration files (`migrations/`) exist but startup uses embedded SQL — migration files are authoritative for production deployments
- No index on `token_hash` (auth_token index used; token_hash lookup is O(n) for large userbases)
- `consumed_at` set but not exposed via API
- Foreign keys on `messages(sender_username, recipient_username)` added as ALTER TABLE (not inline in CREATE TABLE for backward compat)
- Full device integration smoke test requires emulator (`docs/MANUAL_SMOKE_TEST.md`)

## Final Status

All M8 implementation items complete and verified. Not claiming Closed Beta Ready.
