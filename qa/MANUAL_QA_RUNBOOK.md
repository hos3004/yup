# MANUAL_QA_RUNBOOK.md

Status: Draft manual QA guide for next internal build
Project status: Internal Alpha with critical blockers

This runbook is designed for use after Agent 1 finishes M9/M10 stabilization. Do not treat a pass here as Closed Beta readiness. Capture evidence for every step.

## Evidence Folder Convention

For each run, create a local evidence folder outside the repo or under a designated QA artifact location:

```text
qa-evidence/YYYY-MM-DD_build-<commit>/
```

Recommended evidence:

- Terminal logs for server, Flutter, Docker, and tests.
- Screenshots of app states.
- HTTP request/response transcripts.
- PostgreSQL query output.
- Device logs from `flutter logs` or `adb logcat`.
- App build commit and `git status`.

## Environment Setup

### 1. Record Build State

Steps:

1. Open terminal at repo root.
2. Run:

```bash
git status --short --branch
git log --oneline -5
```

Expected result:

- Branch and commit are recorded.
- Working tree state is known.
- Release candidate should be clean except approved QA artifacts.

Evidence to capture:

- Terminal output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

### 2. Start PostgreSQL

Steps:

1. Open terminal at `server/`.
2. Run:

```bash
docker compose up -d postgres
docker compose ps postgres
```

Expected result:

- PostgreSQL container is running.
- Health status is healthy.
- Port 5432 is mapped locally.

Evidence to capture:

- `docker compose ps postgres` output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

### 3. Start Server With DATABASE_URL

Steps:

1. In `server/`, run:

```bash
DATABASE_URL=postgres://yup:yup_dev@localhost:5432/yup?sslmode=disable go run ./cmd/main.go
```

PowerShell equivalent:

```powershell
$env:DATABASE_URL='postgres://yup:yup_dev@localhost:5432/yup?sslmode=disable'
go run ./cmd/main.go
```

Expected result:

- Server starts on configured port, normally 8080.
- Logs indicate PostgreSQL store is selected.
- No migration or connection error occurs.

Evidence to capture:

- Server startup logs.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

### 4. Verify Health Endpoint

Steps:

```bash
curl -i http://127.0.0.1:8080/api/v1/health
```

Expected result:

```json
{"status":"ok"}
```

Evidence to capture:

- Curl output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

### 5. Start Flutter App On Android Emulator

Steps:

1. Start Android emulator.
2. From `yup_mobile/`, run:

```bash
flutter devices
flutter run -d emulator-5556
```

Expected result:

- App builds.
- App installs.
- App opens without startup crash.
- Firebase/push unavailable state must not crash app.

Evidence to capture:

- Build output.
- App screenshot.
- Flutter logs.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Manual Test Flows

## Flow 1: Fresh Install

Steps:

1. Uninstall app from emulator.
2. Clear app data if uninstall is not possible.
3. Install/run current build.
4. Observe first screen.

Expected result:

- No crash.
- App shows registration or restore state.
- No stale account/session data appears.

Evidence to capture:

- Screenshot of first screen.
- Flutter logs from startup.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 2: Register User A

Steps:

1. Enter username `qa_alice_<runid>`.
2. Submit registration.
3. Wait for key generation and key upload.

Expected result:

- User A is registered.
- Auth token is stored locally.
- Public key bundle is uploaded.
- Private key material remains local.

Evidence to capture:

- UI screenshot after registration.
- Server logs.
- Optional DB query showing user and key bundle exist.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 3: Register User B

Steps:

1. Use a second emulator, second app profile, or clear data and register `qa_bob_<runid>`.
2. Preserve User A session separately if possible.

Expected result:

- User B is registered independently.
- User B key bundle exists.
- User A and User B auth tokens are not mixed.

Evidence to capture:

- Screenshot of User B.
- DB query for both users.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 4: Upload Keys Verification

Steps:

1. Verify both users have public identity keys and OTKs in server DB.
2. Fetch B keys as A using API.
3. Fetch B keys again as A using API.

Expected result:

- First key fetch consumes one OTK.
- Second key fetch returns a different OTK or `no_otk_available` if exhausted.
- No private keys are present in server DB.

Evidence to capture:

- Curl outputs.
- SQL query output for OTK consumed state.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 5: A Sends Message To B

Steps:

1. On User A device, open chat with User B.
2. Send: `QA_E2E_SECRET_MESSAGE_<runid>`.
3. Observe local outgoing message state.

Expected result:

- Message appears in A UI.
- Server stores ciphertext only.
- Server logs do not contain plaintext.

Evidence to capture:

- A UI screenshot.
- PostgreSQL query for message row.
- Server logs search for plaintext.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 6: B Fetches Message

Steps:

1. On User B device, open/poll messages.
2. Verify message is displayed in plaintext on B device.
3. Inspect server response if using API proxy/curl.

Expected result:

- B decrypts locally.
- Server response contains ciphertext only.
- ACK is sent after successful processing.

Evidence to capture:

- B UI screenshot.
- Flutter logs.
- Server DB status before/after ACK.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 7: B Replies To A

Steps:

1. On User B device, reply: `QA_REPLY_SECRET_<runid>`.
2. On User A device, fetch/poll.

Expected result:

- A decrypts reply locally.
- Inbound session restoration/reuse works.
- No plaintext appears in server DB or logs.

Evidence to capture:

- UI screenshots for both devices.
- SQL message rows.
- Log search output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 8: Restart Server

Steps:

1. Stop server process.
2. Leave PostgreSQL running.
3. Start server again with same `DATABASE_URL`.
4. Hit health endpoint.

Expected result:

- Server restarts cleanly.
- Migrations do not fail on repeated startup.
- Existing users/messages remain.

Evidence to capture:

- Server stop/start logs.
- Health curl output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 9: Verify Users And Messages Survive Restart

Steps:

1. Query PostgreSQL for users A and B.
2. Query PostgreSQL for key bundles.
3. Query messages by sender/recipient.
4. Open both clients again.

Expected result:

- Users persist.
- Public key bundles persist.
- Pending messages persist until ACK or documented TTL/state transition.
- Clients can resume sessions.

Evidence to capture:

- SQL query output.
- App screenshots.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 10: Verify Unauthenticated Access Fails

Steps:

Run without `Authorization` header:

```bash
curl -i "$BASE_URL/api/v1/messages"
curl -i -X POST "$BASE_URL/api/v1/messages" -H "Content-Type: application/json" -d '{"recipient":"bob","ciphertext":"Y2lwaGVy","message_type":0}'
curl -i -X POST "$BASE_URL/api/v1/devices" -H "Content-Type: application/json" -d '{"token":"abc","platform":"android"}'
```

Expected result:

- All protected endpoints return `401`.

Evidence to capture:

- Curl outputs.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 11: Verify Wrong User Cannot Fetch Queue

Steps:

1. Send A to B.
2. Use A token to call `GET /api/v1/messages`.
3. Use B token to call `GET /api/v1/messages`.

Expected result:

- A does not receive B's message.
- A response is `[]`.
- B receives the message.

Evidence to capture:

- Curl outputs for both tokens.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 12: Verify ACK Behavior

Steps:

1. Send A to B.
2. Capture message ID.
3. Try ACK with A token.
4. Try ACK with B token.
5. Query sent status for A.

Expected result:

- A token ACK fails.
- B token ACK succeeds.
- Sent status updates according to contract.

Evidence to capture:

- Curl outputs.
- SQL status before/after.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 13: Verify Key-Change Warning

Steps:

1. Establish A/B conversation.
2. Record B identity key pinned on A.
3. Re-register or rotate B identity key.
4. A attempts to start/send to B again.

Expected result:

- A detects key change.
- App warns user.
- Silent send is blocked until user verifies or accepts new key.

Evidence to capture:

- Screenshots of warning.
- Local peer key store evidence if available.
- Flutter logs without secrets.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 14: Verify Clear Local Data

Steps:

1. On one device, send/receive at least one message.
2. Use Clear Local Data.
3. Restart app.
4. Inspect local DB existence/passphrase behavior if tooling allows.

Expected result:

- Auth token removed.
- Account pickle removed.
- Session data removed.
- SQLCipher DB deleted or recreated with fresh passphrase.
- Old messages are unrecoverable locally.

Evidence to capture:

- UI screenshots.
- Secure storage test logs if available.
- Local DB file before/after evidence if accessible.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 15: Verify Logout Behavior

Steps:

1. Login/register user.
2. Send/receive a message.
3. Use Logout.
4. Restart app.
5. Attempt restore or re-login according to implemented design.

Expected result:

- Behavior matches documented design.
- If logout preserves encrypted history, auth token must be removed but local encrypted DB may remain.
- If logout is destructive, all local secrets must be cleared.

Evidence to capture:

- Screenshots.
- Storage evidence if available.
- Documentation reference used.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Flow 16: Verify Push Token Registration If Firebase Config Exists

Steps:

1. Confirm `google-services.json` package name matches Android `applicationId`.
2. Start app after registration/auth is available.
3. Confirm `POST /api/v1/devices` succeeds with auth.
4. Query `device_tokens` table.
5. Send A to B and observe push-triggered fetch if FCM credentials exist.

Expected result:

- App does not crash if push unavailable.
- Device token registration happens after auth.
- Token is upserted.
- Push payload contains no plaintext/ciphertext.
- Foreground push triggers fetch.

Evidence to capture:

- Android build output.
- Flutter logs.
- Server logs.
- SQL query output.
- FCM delivery evidence if available.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```
