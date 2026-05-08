# Manual A/B/Restart Smoke Test

> **Purpose:** Validate end-to-end encrypted messaging between two users,
> including session persistence across server restart and key-change detection.
> **Requires:** Two emulators or physical devices, or one device with the ability
> to re-register after clearing data.

## Prerequisites

1. Start PostgreSQL (Docker):
   ```powershell
   cd server
   docker compose up -d
   # Wait for healthy: docker ps --filter "name=postgres" --format "{{.Names}} {{.Status}}"
   ```

2. Start the Go server with PostgreSQL:
   ```powershell
   cd server
   $env:DATABASE_URL="postgres://yup:yup_pass@localhost:5432/yupdb?sslmode=disable"
   go run ./cmd/main.go
   ```
   Expected output: `using PostgresStore` then `YUP server starting on :8080`

3. Build and run the Flutter app on an emulator:
   ```powershell
   cd yup_mobile
   flutter run -d emulator-5554
   ```

## Step 1: Register User A

1. In the Flutter app, enter username `smoke_a` and tap Register.
2. **Verify:** App navigates to chat screen, `smoke_a` displayed in settings.
3. Server check:
   ```powershell
   curl -s http://localhost:8080/api/v1/users/smoke_a | ConvertFrom-Json
   ```
   Expected: `{"username":"smoke_a","created_at":"..."}` (no auth_token in response)

## Step 2: Register User B (Second Device)

1. On the second emulator/device, enter username `smoke_b` and register.
2. **Verify:** App navigates to chat screen, `smoke_b` displayed in settings.

## Step 3: A Sends Message to B

1. In A's app, tap the FAB (+) to start a conversation.
2. Enter recipient `smoke_b` and a test message like `Hello from A!`.
3. Tap send.
4. **Verify:** Message appears in A's chat with status `pending` → `delivered` → `received`.
5. Server check:
   ```powershell
   curl -s http://localhost:8080/api/v1/messages -H "Authorization: Bearer AUTH_TOKEN_A" | ConvertFrom-Json
   ```
   Expected: An envelope with `sender_username: "smoke_a"`, `ciphertext` (no plaintext).

## Step 4: B Fetches and Reads Message

1. In B's app, the poll timer (3s) should pick up the message.
2. **Verify:** B sees the message `Hello from A!` in the chat.
3. Server check (should be empty after fetch):
   ```powershell
   curl -s http://localhost:8080/api/v1/messages -H "Authorization: Bearer AUTH_TOKEN_B" | ConvertFrom-Json
   ```
   Expected: `[]` (empty queue — message was delivered)

## Step 5: B Replies to A

1. In B's app, send a reply: `Hey A, got your message!`.
2. **Verify:** A's app shows the reply after the next poll cycle (≤3s).

## Step 6: Session Persistence After Server Restart

1. Stop the server (Ctrl+C in the server terminal).
2. Restart the server:
   ```powershell
   $env:DATABASE_URL="postgres://yup:yup_pass@localhost:5432/yupdb?sslmode=disable"
   go run ./cmd/main.go
   ```
3. In A's app, send another message: `Still there after restart?`.
4. **Verify:** B receives the message. This validates PostgreSQL persistence
   (users, keys, sessions survive restart).

## Step 7: App Restart (Inbound Session Persistence)

1. Stop the Flutter app (Ctrl+C in the Flutter terminal).
2. Re-launch:
   ```powershell
   flutter run -d emulator-5554
   ```
3. The app should auto-restore the logged-in user (no re-registration needed).
4. B sends a message to A: `Did the app restart work?`
5. **Verify:** A receives the message. This validates that inbound sessions
   are persisted and restored correctly after app restart.

## Step 8: Key Change Scenario

1. Clear data on device A (Settings → Clear Local Data).
2. Re-register `smoke_a` on device A. This generates a new identity key.
3. A sends a message to B.
4. **Verify on B's device:** The key changed warning dialog appears:
   - "The security key for smoke_a has changed..."
   - Options: Cancel, View Verification, Accept New Key
5. Tap "View Verification" — verify the fingerprint is different.
6. Tap "Accept New Key" — the message sends successfully after acceptance.
7. **Verify:** `acceptNewKey` resets the `keyChanged` flag and B's
   `PeerKeyStore` now pins smoke_a's new identity key.

## Step 9: Verification Consistency

1. On both devices, open the verification screen for the conversation.
2. **Verify:** Both show the same "Conversation security fingerprint" value.
3. The fingerprint is order-independent (A→B == B→A).

## Step 10: Cleanup

1. Stop the Flutter apps (Ctrl+C).
2. Stop the server (Ctrl+C).
3. Stop PostgreSQL:
   ```powershell
   cd server
   docker compose down
   ```

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Server refuses connection | PostgreSQL not ready | Wait for `docker ps` to show `healthy` |
| 401 on API calls | Token expired/missing | Check auth header |
| 403 on send | sender_key doesn't match registered key | Clear data and re-register |
| Message not received (stuck pending) | Poll timer not running | Check `startPolling()` was called |
| App crashes on restart | Session pickle corruption | Clear Local Data and re-register |
| Fingerprint mismatch | Different identity keys | Verify both sides using same registered users |
