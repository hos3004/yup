# E2E_CHAT_TEST_SCENARIOS.md

Status: Draft E2E scenario catalog
Project status: Internal Alpha with critical blockers

Use these scenarios after the stabilized build can run on Android and PostgreSQL. Each scenario must produce evidence. No scenario is considered passed without logs/screenshots/query output.

Automation status values:

- Manual Only
- Automatable
- Partially Automatable
- Pending Harness

## Scenario 1: A To B Online

Preconditions:

- PostgreSQL server is running.
- Server is running with `DATABASE_URL`.
- User A and User B are registered on separate app instances or clean profiles.
- Both users have uploaded keys.

Steps:

1. A opens chat with B.
2. A sends `QA_ONLINE_SECRET_<runid>`.
3. B remains online and polling.
4. B receives and decrypts message.
5. B ACKs message.

Expected result:

- A sees outgoing message.
- B sees plaintext locally.
- Server DB stores ciphertext only.
- Sent status updates after ACK.

Data to inspect:

- `messages` table.
- Server logs.
- Flutter logs.
- UI state on both devices.

Evidence required:

- A and B screenshots.
- SQL row for message showing ciphertext only.
- Curl or DB evidence of status transition.

Automation status: Partially Automatable.

## Scenario 2: A To B While B Offline

Preconditions:

- A and B registered.
- B app is closed or network-disabled.

Steps:

1. Stop B app or disable B network.
2. A sends `QA_OFFLINE_SECRET_<runid>`.
3. Inspect server DB before B opens.

Expected result:

- Message remains pending or in documented pre-delivery state.
- Server stores ciphertext only.
- No plaintext appears in logs.

Data to inspect:

- `messages.status`.
- `messages.ciphertext`.
- Server logs.

Evidence required:

- SQL query output.
- Server log search output for plaintext string.

Automation status: Automatable.

## Scenario 3: B Opens Later And Receives

Preconditions:

- Scenario 2 completed.
- B has not fetched the pending message.

Steps:

1. Start B app or restore B network.
2. Wait for polling or push trigger.
3. Confirm B sees plaintext locally.
4. Confirm ACK.

Expected result:

- B receives exactly one copy.
- B decrypts locally.
- Message status changes according to API contract.

Data to inspect:

- B UI.
- `messages.status`.
- Local SQLCipher DB if accessible.

Evidence required:

- Screenshot.
- SQL status before/after.

Automation status: Partially Automatable.

## Scenario 4: Server Restart Before B Opens

Preconditions:

- A sent message to offline B.
- B has not fetched the message.

Steps:

1. Stop server.
2. Restart server with same `DATABASE_URL`.
3. Verify health.
4. Open B app.

Expected result:

- Message is still available to B.
- Users and key bundles persist.
- B decrypts and ACKs.

Data to inspect:

- `users`, `key_bundles`, `messages`.
- Server startup logs.

Evidence required:

- SQL before/after restart.
- Server startup log.
- B screenshot.

Automation status: Automatable.

## Scenario 5: Fetch Without ACK Then App Crash

Preconditions:

- A sends a message to B.
- Test harness can intercept or disable ACK.

Steps:

1. B fetches message.
2. Kill B app before ACK is sent.
3. Restart B app.
4. Fetch again.

Expected result:

- Delivery semantics must match documented contract.
- Message must not be silently lost unless fetch-as-delivery is explicitly accepted and documented.

Data to inspect:

- Message status before fetch.
- Message status after fetch.
- Message status after app restart.

Evidence required:

- SQL status sequence.
- App logs showing crash/kill timing.

Automation status: Pending Harness.

## Scenario 6: Fetch After Restart

Preconditions:

- Message exists in pending or delivered-not-acked state.
- Server has been restarted.

Steps:

1. B calls `GET /api/v1/messages`.
2. B decrypts message.
3. B ACKs message.

Expected result:

- Message availability after restart matches contract.
- No duplicate or lost message.

Data to inspect:

- `messages.status`.
- API response.

Evidence required:

- Curl output.
- SQL output.

Automation status: Automatable.

## Scenario 7: Duplicate Message Prevention

Preconditions:

- A and B registered.
- B has local message storage available.

Steps:

1. A sends one message to B.
2. Force repeated fetch/poll on B.
3. Restart B app and poll again.

Expected result:

- B UI shows one copy.
- Local DB has one row per message ID.
- Server does not reissue ACKed messages.

Data to inspect:

- B UI.
- Local message DB.
- Server message rows.

Evidence required:

- Screenshot.
- Local DB query if available.

Automation status: Partially Automatable.

## Scenario 8: Tampered Ciphertext Rejection

Preconditions:

- A and B registered.
- A valid encrypted message sample exists.

Steps:

1. Modify ciphertext in transit or insert tampered ciphertext through API using valid auth.
2. B fetches message.
3. B attempts decrypt.

Expected result:

- Decryption fails closed.
- App does not display attacker-controlled plaintext.
- Error does not leak private/session keys.
- ACK behavior after decrypt failure must be documented.

Data to inspect:

- Flutter logs.
- Server DB row.
- B UI.

Evidence required:

- Tampered payload.
- B error state screenshot/log.

Automation status: Pending Harness.

## Scenario 9: Sender Spoofing Rejection

Preconditions:

- A, B, and C registered.
- A has uploaded keys.

Steps:

1. Use A token to call `POST /api/v1/messages`.
2. Include spoofed sender fields if client/API still accepts them.
3. Send to B.
4. Inspect stored envelope.

Expected result:

- Stored `sender_username` is A.
- Stored `sender_curve_key` is A's registered curve key.
- Client-supplied sender fields are ignored or rejected.

Data to inspect:

- API response.
- `messages.sender_username`.
- `messages.sender_curve_key`.

Evidence required:

- Curl request/response.
- SQL output.

Automation status: Automatable.

## Scenario 10: Wrong User Queue Fetch Rejection

Preconditions:

- A sends message to B.
- C is registered.

Steps:

1. Fetch with A token.
2. Fetch with C token.
3. Fetch with B token.

Expected result:

- A and C do not receive B's message.
- B receives message.
- Empty queues return `[]`, not `null`.

Data to inspect:

- API responses.
- Server logs.

Evidence required:

- Curl output for all three users.

Automation status: Automatable.

## Scenario 11: Key Changed Warning

Preconditions:

- A has pinned B's identity key from an earlier conversation.

Steps:

1. Rotate or re-register B to create a new identity key.
2. A attempts to open/send conversation to B.

Expected result:

- A detects identity key change.
- Warning appears.
- Silent send is blocked.

Data to inspect:

- Peer key store.
- UI warning state.
- Flutter logs.

Evidence required:

- Warning screenshot.
- Stored old/new key evidence without private key material.

Automation status: Partially Automatable.

## Scenario 12: Re-Verify After Key Change

Preconditions:

- Scenario 11 completed.

Steps:

1. User chooses verification path.
2. Compare fingerprints out of band.
3. Accept new key only after verification.
4. Send new message.

Expected result:

- New key is accepted only through explicit user action.
- Warning state clears.
- New conversation can proceed.

Data to inspect:

- Peer key store.
- Verification UI.

Evidence required:

- Verification screenshot.
- Message send after accept.

Automation status: Manual Only.

## Scenario 13: Clear Local Data And Re-Register

Preconditions:

- User has sent/received messages.

Steps:

1. Clear local data.
2. Restart app.
3. Re-register same or new username according to server policy.
4. Inspect old local data.

Expected result:

- Local auth token, account pickle, sessions, identity keys, DB passphrase, and messages are removed.
- Old local DB cannot be opened with old key.

Data to inspect:

- Secure storage keys if test hooks exist.
- SQLCipher DB file.
- UI state.

Evidence required:

- Screenshot before/after.
- Storage evidence.

Automation status: Partially Automatable.

## Scenario 14: Logout And Return

Preconditions:

- User is registered and has local encrypted history.

Steps:

1. Logout.
2. Restart app.
3. Return using documented restore/login path.

Expected result:

- Behavior matches documented design.
- Auth token handling is safe.
- Local encrypted history handling is explicit and consistent.

Data to inspect:

- Secure storage keys.
- Local DB.
- UI state.

Evidence required:

- Screenshots.
- Storage evidence if available.

Automation status: Partially Automatable.

## Scenario 15: No Plaintext In Server Database

Preconditions:

- Known unique plaintext sent: `QA_NO_PLAINTEXT_<runid>`.

Steps:

1. Send message containing unique plaintext.
2. Query server DB tables for that string.
3. Inspect message rows.

Expected result:

- Plaintext string does not appear in PostgreSQL.
- Only ciphertext is stored.

Data to inspect:

- `messages`.
- Any JSON/session/device tables.

Evidence required:

- SQL query output showing no plaintext match.

Automation status: Automatable.

## Scenario 16: No Plaintext In Logs

Preconditions:

- Server and Flutter logging enabled.
- Known unique plaintext sent.

Steps:

1. Send `QA_LOG_SECRET_<runid>`.
2. Collect server logs.
3. Collect Flutter logs.
4. Search for plaintext, auth token, private keys, session keys, and pickles.

Expected result:

- Plaintext absent.
- Tokens absent or redacted.
- Private/session keys absent.

Data to inspect:

- Server stdout/stderr logs.
- Flutter logs.
- Android logcat.

Evidence required:

- Search command and output.

Automation status: Automatable.
