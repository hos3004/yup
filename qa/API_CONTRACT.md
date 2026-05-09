# API_CONTRACT.md

Status: Draft API contract for QA verification
Project status: Internal Alpha with critical blockers
Scope: Server HTTP API used by the Flutter client

This document is not a final API specification. It records the intended contract to verify after M9/M10 stabilization. Any item marked Pending Verification must be confirmed against the stabilized implementation before release acceptance.

Base URL examples:

- Android emulator to local host: `http://10.0.2.2:8080`
- Local desktop curl: `http://127.0.0.1:8080`

Common requirements:

- Request and response format is JSON unless noted.
- Authenticated endpoints use `Authorization: Bearer <auth_token>`.
- The server must derive the authenticated username from the bearer token, not from client-supplied sender fields.
- Server must never accept plaintext message bodies for message send APIs.
- Server must never return private keys, account pickles, session pickles, SQLCipher passphrases, or Rust crypto state.

Common error shape:

```json
{
  "error": "human_readable_error"
}
```

## POST /api/v1/users

Purpose: Register a new username and receive an auth token.

Auth: Public.

Request body:

```json
{
  "username": "alice"
}
```

Response body:

```json
{
  "username": "alice",
  "auth_token": "<opaque bearer token>",
  "created_at": "2026-05-09T12:00:00Z"
}
```

Status codes:

- `201 Created`: user registered.
- `400 Bad Request`: invalid JSON or invalid username.
- `409 Conflict`: username already exists.
- `405 Method Not Allowed`: wrong method.

Validation rules:

- `username` is required.
- Username length: 3 to 32 characters.
- Allowed characters: ASCII letters, ASCII digits, `_`, `-`.
- Leading/trailing spaces should be trimmed or rejected consistently.

Security notes:

- Auth token must be generated with cryptographically secure randomness.
- Auth token must not be logged.
- PostgreSQL storage of auth tokens must be verified. Plaintext token storage is not acceptable for release unless explicitly risk-accepted.
- `GET /api/v1/users/{username}` is not in this contract list; if present, it must not expose `auth_token`.

Example curl request:

```bash
curl -i -X POST "$BASE_URL/api/v1/users" \
  -H "Content-Type: application/json" \
  -d '{"username":"alice"}'
```

Expected success example:

```http
HTTP/1.1 201 Created
Content-Type: application/json

{"username":"alice","auth_token":"<token>","created_at":"2026-05-09T12:00:00Z"}
```

Expected failure examples:

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{"error":"username must be 3-32 characters"}
```

```http
HTTP/1.1 409 Conflict
Content-Type: application/json

{"error":"username already exists"}
```

## PUT /api/v1/keys

Status: Pending Verification.

Purpose: Upload the authenticated user's public identity keys and one-time keys.

Auth: Required.

Important route note:

- Historical implementations used `PUT /api/v1/keys/{username}`.
- The intended stabilized contract requested for QA is `PUT /api/v1/keys`.
- QA must verify which route Agent 1 stabilizes and record any mismatch as an API contract defect.

Request body:

```json
{
  "curve_key": "<base64 public Curve25519 identity key>",
  "ed_key": "<base64 public Ed25519 identity key>",
  "one_time_keys": [
    "<base64 one-time key>"
  ],
  "signature": "<optional base64 signature>"
}
```

Response body:

```json
{
  "device_id": "<server generated device id>",
  "curve_key": "<base64 public Curve25519 identity key>",
  "ed_key": "<base64 public Ed25519 identity key>",
  "one_time_keys": [
    "<base64 one-time key>"
  ],
  "signature": "<optional base64 signature>"
}
```

Status codes:

- `200 OK`: keys uploaded.
- `400 Bad Request`: invalid key bundle, invalid base64, too many OTKs, user not found.
- `401 Unauthorized`: missing or invalid bearer token.
- `405 Method Not Allowed`: wrong method.

Validation rules:

- `curve_key` and `ed_key` are required.
- Keys must be valid base64 as defined by the server.
- Public key lengths must match stabilized server validation.
- `one_time_keys` must be an array.
- One-time key count limit must be verified after stabilization.
- Server must ignore any username in the body and bind upload to the auth token.

Security notes:

- Only public keys may be uploaded.
- Private keys, account pickles, session pickles, and SQLCipher passphrases must never be sent.
- Re-upload semantics must be verified: whether old OTKs are replaced, appended, or rejected.

Example curl request:

```bash
curl -i -X PUT "$BASE_URL/api/v1/keys" \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"curve_key":"YUtleQ==","ed_key":"YUVkS2V5","one_time_keys":["YU90azE="]}'
```

Expected success example:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"device_id":"<id>","curve_key":"YUtleQ==","ed_key":"YUVkS2V5","one_time_keys":["YU90azE="]}
```

Expected failure examples:

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"error":"missing or invalid token"}
```

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{"error":"invalid key format"}
```

## GET /api/v1/keys/{username}

Purpose: Fetch a peer's public key bundle and consume one available one-time key.

Auth: Required.

Request body: none.

Response body:

```json
{
  "device_id": "<device id>",
  "curve_key": "<base64 public Curve25519 identity key>",
  "ed_key": "<base64 public Ed25519 identity key>",
  "one_time_keys": [
    "<single consumed one-time key>"
  ],
  "no_otk_available": true
}
```

Status codes:

- `200 OK`: key bundle returned.
- `401 Unauthorized`: missing or invalid bearer token.
- `404 Not Found`: user or key bundle not found.
- `405 Method Not Allowed`: wrong method.

Validation rules:

- `{username}` must identify an existing user with uploaded keys.
- On each successful fetch, at most one OTK should be returned.
- Once returned, an OTK must not be returned again.
- If no OTK is available, `one_time_keys` must be `[]` and the response should include `no_otk_available: true`.

Security notes:

- This endpoint exposes public identity keys and OTKs only.
- OTK consumption must be atomic under concurrent requests.
- The server must not expose private keys or session state.

Example curl request:

```bash
curl -i "$BASE_URL/api/v1/keys/bob" \
  -H "Authorization: Bearer $ALICE_TOKEN"
```

Expected success example:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"device_id":"<id>","curve_key":"YktleQ==","ed_key":"YkVkS2V5","one_time_keys":["Yk90azE="]}
```

Expected failure examples:

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"error":"missing or invalid token"}
```

```http
HTTP/1.1 404 Not Found
Content-Type: application/json

{"error":"keys not found for user"}
```

## POST /api/v1/messages

Purpose: Store an encrypted message envelope for a recipient.

Auth: Required.

Request body:

```json
{
  "recipient": "bob",
  "ciphertext": "<base64 ciphertext>",
  "message_type": 0
}
```

Response body:

```json
{
  "id": "<message id>",
  "sender_username": "alice",
  "ciphertext": "<base64 ciphertext>",
  "message_type": 0,
  "sender_curve_key": "<sender registered public curve key>",
  "status": "pending",
  "created_at": "2026-05-09T12:00:00Z"
}
```

Status codes:

- `201 Created`: encrypted envelope stored.
- `400 Bad Request`: invalid recipient, ciphertext, message type, missing sender key bundle, or recipient not found.
- `401 Unauthorized`: missing or invalid bearer token.
- `405 Method Not Allowed`: wrong method.

Validation rules:

- `recipient` is required and must be a valid username.
- `ciphertext` is required and must be base64.
- `message_type` allowed values must be verified. Historical values include Olm pre-key/message types.
- Server must derive `sender_username` and `sender_curve_key` from authenticated server-side data.
- Client-supplied `sender`, `sender_username`, or `sender_key` must not be trusted.

Security notes:

- Plaintext must never be accepted or stored.
- Server cannot decrypt ciphertext.
- Sender spoofing must fail.
- Message body size must be limited.
- Push notification payload, if sent, must not include plaintext or ciphertext.

Example curl request:

```bash
curl -i -X POST "$BASE_URL/api/v1/messages" \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"recipient":"bob","ciphertext":"Y2lwaGVydGV4dA==","message_type":0}'
```

Expected success example:

```http
HTTP/1.1 201 Created
Content-Type: application/json

{"id":"<id>","sender_username":"alice","ciphertext":"Y2lwaGVydGV4dA==","message_type":0,"sender_curve_key":"<alice curve key>","status":"pending","created_at":"2026-05-09T12:00:00Z"}
```

Expected failure examples:

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"error":"missing or invalid token"}
```

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{"error":"recipient not found"}
```

## GET /api/v1/messages

Purpose: Fetch pending encrypted message envelopes for the authenticated user.

Auth: Required.

Request body: none.

Response body:

```json
[
  {
    "id": "<message id>",
    "sender_username": "alice",
    "ciphertext": "<base64 ciphertext>",
    "message_type": 0,
    "sender_curve_key": "<sender public curve key>",
    "status": "delivered",
    "created_at": "2026-05-09T12:00:00Z"
  }
]
```

Empty queue response:

```json
[]
```

Status codes:

- `200 OK`: messages returned, possibly empty.
- `401 Unauthorized`: missing or invalid bearer token.
- `405 Method Not Allowed`: wrong method.

Validation rules:

- Must return only messages where the authenticated user is the recipient.
- Must return `[]`, not `null`, for empty queues.
- Fetch side effects must be documented and verified: pending to delivered, delete-on-fetch, or no state change.

Security notes:

- Wrong user must not fetch another user's queue.
- Server response must contain ciphertext only.
- Fetch without ACK crash behavior is critical and must be tested.

Example curl request:

```bash
curl -i "$BASE_URL/api/v1/messages" \
  -H "Authorization: Bearer $BOB_TOKEN"
```

Expected success example:

```http
HTTP/1.1 200 OK
Content-Type: application/json

[]
```

Expected failure example:

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"error":"missing or invalid token"}
```

## GET /api/v1/messages/sent

Purpose: Return message envelopes sent by the authenticated user with current server status.

Auth: Required.

Request body: none.

Response body:

```json
[
  {
    "id": "<message id>",
    "sender_username": "alice",
    "ciphertext": "<base64 ciphertext>",
    "message_type": 0,
    "sender_curve_key": "<sender public curve key>",
    "status": "received",
    "created_at": "2026-05-09T12:00:00Z"
  }
]
```

Empty response:

```json
[]
```

Status codes:

- `200 OK`: sent messages returned, possibly empty.
- `401 Unauthorized`: missing or invalid bearer token.
- `405 Method Not Allowed`: wrong method.

Validation rules:

- Must return only messages where authenticated user is the sender.
- Must return `[]`, not `null`, when no sent messages exist.
- Status values must be documented after stabilization. Historical values include `pending`, `delivered`, `received`.

Security notes:

- Must not expose messages sent by another user.
- Must not leak recipient queues.

Example curl request:

```bash
curl -i "$BASE_URL/api/v1/messages/sent" \
  -H "Authorization: Bearer $ALICE_TOKEN"
```

Expected success example:

```http
HTTP/1.1 200 OK
Content-Type: application/json

[]
```

Expected failure example:

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"error":"missing or invalid token"}
```

## POST /api/v1/messages/{messageID}/ack

Purpose: Acknowledge that the authenticated recipient processed a message.

Auth: Required.

Request body: none.

Response body:

```json
{
  "status": "acknowledged"
}
```

Status codes:

- `200 OK`: ACK accepted.
- `400 Bad Request`: message not found, wrong recipient, invalid status transition.
- `401 Unauthorized`: missing or invalid bearer token.
- `405 Method Not Allowed`: wrong method.

Validation rules:

- `{messageID}` is required.
- Only the authenticated recipient can ACK.
- Wrong sender or unrelated user must fail.
- ACK must be idempotency behavior Pending Verification: repeated ACK may succeed or fail, but must be documented.

Security notes:

- Sender must not be able to ACK recipient delivery.
- ACK must not expose message plaintext.
- ACK status transitions must be consistent across InMemory and PostgreSQL stores.

Example curl request:

```bash
curl -i -X POST "$BASE_URL/api/v1/messages/$MESSAGE_ID/ack" \
  -H "Authorization: Bearer $BOB_TOKEN"
```

Expected success example:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"acknowledged"}
```

Expected failure examples:

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{"error":"not the recipient of this message"}
```

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"error":"missing or invalid token"}
```

## POST /api/v1/devices

Purpose: Register or update a push notification device token for the authenticated user.

Auth: Required.

Request body:

```json
{
  "token": "<push provider device token>",
  "platform": "android"
}
```

Response body:

```json
{
  "status": "registered"
}
```

Status codes:

- `200 OK`: token registered or updated.
- `400 Bad Request`: invalid body, token, or platform.
- `401 Unauthorized`: missing or invalid bearer token.
- `405 Method Not Allowed`: wrong method.
- `500 Internal Server Error`: persistence failure.

Validation rules:

- `token` is required.
- Token maximum length must be verified after stabilization.
- `platform` allowed values Pending Verification. Historical values include `android`, `ios`, `web`.
- Duplicate token for same user should upsert, not duplicate rows.

Security notes:

- Device token is sensitive operational metadata.
- Push payload must not include plaintext, ciphertext, auth tokens, private keys, session keys, or account pickles.
- Registration must occur only after auth token is available.
- Token refresh handling must update the server.
- Invalid token cleanup is Pending Verification.

Example curl request:

```bash
curl -i -X POST "$BASE_URL/api/v1/devices" \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"token":"fcm_token_example","platform":"android"}'
```

Expected success example:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"registered"}
```

Expected failure examples:

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{"error":"missing or invalid token"}
```

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{"error":"invalid token"}
```

## GET /api/v1/health

Purpose: Lightweight server health check.

Auth: Public.

Request body: none.

Response body:

```json
{
  "status": "ok"
}
```

Status codes:

- `200 OK`: server process is responding.

Validation rules:

- Must not require auth.
- Must not expose secrets, database DSNs, build metadata, tokens, or internal errors.

Security notes:

- This endpoint proves process liveness only.
- It must not be used as proof that PostgreSQL, FCM, Rust crypto, or migrations are healthy unless explicitly extended and documented.

Example curl request:

```bash
curl -i "$BASE_URL/api/v1/health"
```

Expected success example:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

Expected failure examples:

```text
Connection refused: server is not running or port is wrong.
```
