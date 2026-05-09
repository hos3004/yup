# M8.1 — PostgreSQL Semantics Hardening Report

## Changes Applied

### Fix 1: Empty List Responses
- **File**: `server/internal/service/postgres_store.go`
- **Change**: `GetPendingEnvelopes` and `GetSentMessages` now return `make([]*model.Envelope, 0)` instead of `var envs []*model.Envelope` (which is nil).
- **Result**: JSON responses always `[]` not `null` for empty lists.

### Fix 2: Shared Handler Test Suite
- **File**: `server/internal/handler/handler_store_test.go` (new)
- **Change**: Created `runStoreTests()` that runs identical expectations against any `DataStore`. Called by `TestStoreSuite_InMemory` and `TestStoreSuite_Postgres`.
- **Result**: 6 shared tests covering register+get, token validation, key upload+OTK consumption, send message lifecycle, and sent messages. Both stores must pass the same assertions.

### Fix 3: Delivery State Machine
- **Files**: `server/internal/service/postgres_store.go`, `store.go`
- **Change**:
  - `GetPendingEnvelopes`: queries `status = 'pending'`, returns results, marks as `'delivered'`
  - `AckMessage`: transitions `'pending'` or `'delivered'` → `'received'`
  - `NewPostgresStore`: startup resets `delivered`→`pending` (retry after restart)
- **Result**: Messages stay pending until fetched (one visible fetch per session). ACK moves to received. Restart retries un-ACK'd messages.

### Fix 4: Sender Key Binding (Option A)
- **File**: `server/internal/handler/handler.go`
- **Change**: Removed `sender_key` from the client request body. Server always derives `sender_curve_key` from the authenticated user's registered key bundle via `GetCurveKey`.
- **Result**: Client no longer controls sender_key. Send fails with 400 if sender hasn't uploaded keys. Sender spoofing attack surface eliminated.

### Fix 5: OTK Schema + Row-ID Consumption
- **Files**: `migrations/000002_semantics_hardening.up.sql`, `server/internal/service/postgres_store.go`
- **Change**:
  - Added `consumed_at TIMESTAMPTZ` column to `one_time_keys`
  - Added `UNIQUE(username, key_value)` constraint
  - `GetKeyBundle`: now selects `id, key_value` and updates by `id` with `consumed = TRUE, consumed_at = NOW()`
  - Updated index: `idx_otk_username_consumed ON (username, consumed, consumed_at)`
- **Result**: OTK consumption is row-id precise with timestamp. Unique constraint prevents duplicate key uploads.

### Fix 6: Message Foreign Keys
- **Files**: `migrations/000002_semantics_hardening.up.sql`, embedded SQL in `postgres_store.go`
- **Change**: Added `fk_messages_sender` and `fk_messages_recipient` foreign keys on `messages(sender_username, recipient_username)` → `users(username) ON DELETE CASCADE`
- **Result**: Cascading deletes for user removal. Referential integrity enforced.

### Fix 7: Token Storage (SHA-256 Hashing)
- **File**: `server/internal/service/postgres_store.go`
- **Change**:
  - `RegisterUser`: stores `sha256(token)` in `token_hash` column (raw token kept in `auth_token` for backward compat)
  - `ValidateToken`: hashes incoming token, queries by `token_hash` via `SELECT username FROM users WHERE token_hash = $1`
- **Result**: Server no longer stores plaintext-equivalent tokens in the primary lookup column. Hash comparison is constant-time (no `subtle.ConstantTimeCompare` needed since hash output is fixed-length).

### Fix 8: Migration Policy
- **File**: `server/Makefile`
- **Change**: Fixed `db-migrate` target — corrected database URL (`yup_pass`@`yupdb`). Fixed `test-integration` target — removed non-existent `-tags=integration`, added `TestStoreSuite_Postgres` to pattern.
- **Result**: `make db-migrate CMD="up"` and `make test-integration` work correctly.

## Test Results

| Suite | Count | Status |
|-------|-------|--------|
| Handler tests (InMemory) | 32 | All PASS |
| Shared store suite (InMemory) | 6 | All PASS |
| Shared store suite (Postgres) | 6 | All PASS |
| PG integration tests | 20 | All PASS |
| **Total** | **32 (unique)** + **20 PG** | **All PASS** |
| `go vet ./...` | — | Clean |

## Remaining Blockers

- **Rust MSVC build**: `msvcrt.lib` missing from VS 2022 Community; using `cargo +stable-gnu` workaround
- **Full device integration smoke test**: requires emulator (manual script at `docs/MANUAL_SMOKE_TEST.md`)
- No index on `token_hash` column (OK for current userbase scale)

## Final Status

All 10 M8.1 audit items resolved. Both `InMemoryStore` and `PostgresStore` produce identical API responses. PostgreSQL persistence is semantically equivalent and hardened. Not claiming Closed Beta Ready.
