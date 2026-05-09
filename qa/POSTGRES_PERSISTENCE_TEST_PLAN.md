# POSTGRES_PERSISTENCE_TEST_PLAN.md

Status: Draft PostgreSQL persistence QA plan
Project status: Internal Alpha with critical blockers

Scope: PostgreSQL behavior only. This plan does not validate Flutter UI, Rust crypto correctness, or push delivery except where they affect persisted state.

## Setup

Start database:

```bash
cd server
docker compose up -d postgres
docker compose ps postgres
```

Start server:

```bash
DATABASE_URL=postgres://yup:yup_dev@localhost:5432/yup?sslmode=disable go run ./cmd/main.go
```

Set shell helper:

```bash
export BASE_URL=http://127.0.0.1:8080
```

Run psql:

```bash
docker exec -it server-postgres-1 psql -U yup -d yup
```

## SQL Inspection Queries

List tables:

```sql
\dt
```

Inspect users:

```sql
SELECT username, created_at FROM users ORDER BY created_at DESC LIMIT 20;
```

Check auth token storage:

```sql
SELECT username,
       auth_token IS NOT NULL AND auth_token <> '' AS has_plaintext_auth_token,
       token_hash IS NOT NULL AND token_hash <> '' AS has_token_hash
FROM users
ORDER BY created_at DESC
LIMIT 20;
```

Inspect key bundles:

```sql
SELECT username, device_id, curve_key, ed_key, updated_at
FROM key_bundles
ORDER BY updated_at DESC
LIMIT 20;
```

Inspect OTK lifecycle:

```sql
SELECT username, key_value, consumed, consumed_at, created_at
FROM one_time_keys
WHERE username IN ('qa_alice', 'qa_bob')
ORDER BY username, id;
```

Inspect messages:

```sql
SELECT id, sender_username, recipient_username, ciphertext, message_type,
       sender_curve_key, status, created_at, delivered_at
FROM messages
ORDER BY created_at DESC
LIMIT 20;
```

Search for plaintext test string:

```sql
SELECT 'messages' AS table_name, id
FROM messages
WHERE ciphertext LIKE '%QA_SECRET_STRING%';
```

Check private/session key columns:

```sql
SELECT table_name, column_name
FROM information_schema.columns
WHERE table_schema = 'public'
  AND (
    column_name ILIKE '%private%'
    OR column_name ILIKE '%session%'
    OR column_name ILIKE '%pickle%'
    OR column_name ILIKE '%passphrase%'
  )
ORDER BY table_name, column_name;
```

Check empty queue API response:

```bash
curl -s "$BASE_URL/api/v1/messages" -H "Authorization: Bearer $TOKEN"
curl -s "$BASE_URL/api/v1/messages/sent" -H "Authorization: Bearer $TOKEN"
```

Expected response for empty lists:

```json
[]
```

## Test P1: Users Persist After Server Restart

Purpose: Verify registered users survive server process restart.

Steps:

1. Register User A and User B.
2. Query `users`.
3. Stop server.
4. Start server with same `DATABASE_URL`.
5. Query `users` again.

Expected result:

- Both users remain present.
- No duplicate rows.
- Server startup does not destroy data.

Evidence:

- Registration responses.
- SQL before/after restart.
- Server restart logs.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P2: Key Bundles Persist

Purpose: Verify public key bundles survive restart.

Steps:

1. Upload key bundles for A and B.
2. Query `key_bundles`.
3. Restart server.
4. Fetch B key bundle as A.

Expected result:

- Public identity keys persist.
- One OTK is returned and consumed.
- No private keys are present.

Evidence:

- SQL query output.
- API response.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P3: One-Time Keys Consumed Once

Purpose: Verify OTKs are not reused.

Steps:

1. Upload at least two OTKs for B.
2. Fetch B keys as A.
3. Fetch B keys as A again.
4. Query `one_time_keys`.

Expected result:

- First and second fetch return different OTKs.
- Consumed OTKs have `consumed = true`.
- Consumed OTKs are not returned again.

Evidence:

- API response 1 and 2.
- SQL query output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P4: consumed_at Behavior If Available

Purpose: Verify `consumed_at` is populated when the schema supports it.

Steps:

1. Confirm `one_time_keys.consumed_at` exists.
2. Fetch an OTK.
3. Query consumed row.

Expected result:

- `consumed_at` is non-null for consumed OTK.
- `consumed_at` remains null for unconsumed OTKs.

Evidence:

- `information_schema.columns` output.
- OTK query output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P5: Messages Persist While Pending

Purpose: Verify offline delivery persistence.

Steps:

1. A sends message to B while B is offline.
2. Query `messages`.
3. Restart server.
4. Query `messages`.
5. B fetches messages.

Expected result:

- Message remains present before B fetches.
- Message remains after restart.
- B fetches message once.

Evidence:

- SQL output before/after restart.
- B fetch API response.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P6: ACK Updates State

Purpose: Verify recipient ACK changes message status.

Steps:

1. A sends to B.
2. B fetches message.
3. Query status.
4. B ACKs message.
5. Query status.

Expected result:

- Status transitions match documented contract.
- Wrong user ACK fails.
- Recipient ACK succeeds.

Evidence:

- Curl outputs.
- SQL status outputs.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P7: Empty Queues Return []

Purpose: Prevent client JSON cast crashes.

Steps:

1. Register a user with no pending messages.
2. Call `GET /api/v1/messages`.
3. Call `GET /api/v1/messages/sent`.

Expected result:

- Both responses are `[]`.
- Neither response is `null`.

Evidence:

- Raw curl output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P8: No Null List Responses

Purpose: Confirm all list APIs return arrays.

Steps:

1. Exercise empty and non-empty list endpoints.
2. Capture raw responses.
3. Parse as JSON arrays.

Expected result:

- Empty list endpoints return `[]`.
- Non-empty list endpoints return `[ ... ]`.
- No endpoint returns JSON `null` for list type.

Evidence:

- Curl output.
- JSON parser output if automated.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P9: No Plaintext Message Storage

Purpose: Verify server stores ciphertext only.

Steps:

1. Send message with unique plaintext `QA_DB_SECRET_<runid>`.
2. Query all text-like message fields for that value.
3. Inspect `messages.ciphertext`.

Expected result:

- Plaintext string is absent from DB.
- Ciphertext is present and not equal to plaintext.

Evidence:

- SQL search output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P10: No Private Or Session Keys Stored

Purpose: Verify DB schema stores public relay data only.

Steps:

1. Run information schema query for sensitive column names.
2. Inspect table contents for obvious pickles/session keys.

Expected result:

- No private key, account pickle, session pickle, or SQLCipher passphrase columns.
- Public key columns are acceptable.

Evidence:

- Schema query output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P11: Auth Tokens Not Stored Plaintext If Fixed

Purpose: Verify stabilization removed plaintext auth token storage.

Steps:

1. Register user.
2. Query `users` token-related columns.
3. Confirm validation still works through API.

Expected result:

- No plaintext bearer token column is populated.
- Hash or equivalent verifier is present.
- API auth still validates token.

Evidence:

- SQL output.
- Authenticated API response.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P12: Migration Idempotency

Purpose: Verify repeated startup does not fail or damage data.

Steps:

1. Start server with empty DB.
2. Stop server.
3. Start server again.
4. Register user and send message.
5. Start server a third time.

Expected result:

- No migration error.
- No destructive reset.
- Existing data remains.

Evidence:

- Startup logs.
- SQL row counts.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P13: Docker Compose Startup

Purpose: Verify local database starts reliably.

Steps:

```bash
docker compose down
docker compose up -d postgres
docker compose ps postgres
docker compose logs postgres --tail=100
```

Expected result:

- PostgreSQL reaches healthy state.
- No crash loop.

Evidence:

- Compose output.
- Logs.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test P14: Test Database Isolation

Purpose: Verify `DATABASE_URL_TEST=... go test ./...` is reliable.

Steps:

1. Run:

```bash
DATABASE_URL_TEST=postgres://yup:yup_dev@localhost:5432/yup?sslmode=disable go test ./... -v -count=1
```

2. Run it three times.
3. If failure occurs, run with `-p 1` to compare.

Expected result:

- Required default command passes repeatedly.
- Tests do not clobber each other across packages.

Evidence:

- Full command outputs.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```
