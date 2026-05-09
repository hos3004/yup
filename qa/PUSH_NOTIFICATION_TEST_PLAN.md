# PUSH_NOTIFICATION_TEST_PLAN.md

Status: Draft push notification QA plan
Project status: Internal Alpha with critical blockers

Scope: Push notification foundation only. Do not claim push is complete unless the stabilized implementation proves each item below.

## Prerequisites

Firebase configuration:

- Firebase project exists.
- Android app is registered in Firebase.
- `google-services.json` exists at `yup_mobile/android/app/google-services.json`.
- `google-services.json` package name matches Android `applicationId`.
- Server-side service account JSON exists for FCM if testing real delivery.
- `GOOGLE_APPLICATION_CREDENTIALS` points to the service account JSON when testing real push.

Local server:

```bash
cd server
docker compose up -d postgres
DATABASE_URL=postgres://yup:yup_dev@localhost:5432/yup?sslmode=disable go run ./cmd/main.go
```

Flutter:

```bash
cd yup_mobile
flutter pub get
flutter build apk --debug --target-platform android-x64
flutter run -d emulator-5556
```

## Test PN1: Firebase Package Name Matches applicationId

Purpose: Prevent Android build failure.

Steps:

1. Read `yup_mobile/android/app/build.gradle.kts`.
2. Record `applicationId`.
3. Read `yup_mobile/android/app/google-services.json`.
4. Record `client[].client_info.android_client_info.package_name`.
5. Build Android APK.

Expected result:

- Package names match exactly.
- Android build passes.

Evidence:

- File snippets.
- Build output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test PN2: App Builds Without Crashing If Push Unavailable

Purpose: Push must not be a startup blocker.

Steps:

1. Remove or withhold server FCM credentials.
2. Ensure Firebase client config is valid or test documented unavailable mode.
3. Launch app.
4. Observe startup before and after registration.

Expected result:

- App opens.
- Missing push availability does not prevent `runApp`.
- User can register and message without push.

Evidence:

- Flutter logs.
- Startup screenshot.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test PN3: Push Registration Only After Auth

Purpose: Device token registration endpoint requires auth and must not be called before auth exists.

Steps:

1. Clear app data.
2. Start app and capture server logs.
3. Register user.
4. Capture API call to `POST /api/v1/devices`.

Expected result:

- No unauthenticated `/devices` call before registration/session restore.
- After auth token exists, `/devices` returns `200`.
- If token retrieval fails, app continues.

Evidence:

- Server logs.
- HTTP trace.
- Flutter logs.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test PN4: Device Token Upsert

Purpose: Re-registering same token should update, not duplicate.

Steps:

1. Register user.
2. POST device token `token_A` for platform `android`.
3. POST same token again.
4. Query `device_tokens`.

SQL:

```sql
SELECT username, token, platform, created_at, updated_at
FROM device_tokens
WHERE username = 'qa_alice'
ORDER BY updated_at DESC;
```

Expected result:

- One row for `(username, token)`.
- `updated_at` changes on second registration.

Evidence:

- Curl outputs.
- SQL output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test PN5: Auth Required For Device Registration

Purpose: Ensure anonymous clients cannot attach tokens to accounts.

Steps:

```bash
curl -i -X POST "$BASE_URL/api/v1/devices" \
  -H "Content-Type: application/json" \
  -d '{"token":"qa_token","platform":"android"}'
```

Expected result:

- `401 Unauthorized`.

Evidence:

- Curl output.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test PN6: No Plaintext In Push Payload

Purpose: Verify push payload does not weaken E2EE.

Steps:

1. Configure a mock or instrumented notifier if available.
2. Send message with plaintext `QA_PUSH_SECRET_<runid>`.
3. Capture push payload.

Expected result:

- Payload contains routing metadata only, such as type and sender.
- Payload does not contain plaintext.
- Payload does not contain ciphertext unless explicitly risk-accepted.
- Payload does not contain auth token, private key, session key, or pickle.

Evidence:

- Captured payload.
- Server logs.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test PN7: Token Refresh Handling

Purpose: Verify refreshed FCM tokens update the server.

Steps:

1. Register device token.
2. Trigger token refresh if possible or use test hook/mock.
3. Observe app POST new token.
4. Query `device_tokens`.

Expected result:

- New token is registered.
- Old token handling is documented: retained or removed.
- App does not lose auth state.

Evidence:

- Flutter logs.
- Server logs.
- SQL query.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test PN8: Foreground Notification Handling

Purpose: Verify foreground push triggers immediate fetch.

Steps:

1. A and B registered.
2. B app open in foreground.
3. A sends message to B.
4. Observe B logs and UI.

Expected result:

- B receives push.
- Push trigger starts or accelerates fetch.
- B displays decrypted message once.

Evidence:

- B Flutter logs.
- B screenshot.
- Server message status.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test PN9: Background Notification Handling

Status: Pending if not implemented.

Purpose: Verify message fetch or user-visible behavior when app is backgrounded.

Steps:

1. Put B app in background.
2. A sends message to B.
3. Observe notification and app behavior.
4. Bring B app foreground.

Expected result:

- If implemented: B handles background message according to documented design.
- If not implemented: limitation is documented and not claimed complete.

Evidence:

- Android notification screenshot.
- Flutter/adb logs.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test PN10: Resume/Open Notification Handling

Status: Pending if not implemented.

Purpose: Verify tapping/opening notification leads to correct fetch/navigation.

Steps:

1. Background or terminate B app.
2. A sends message to B.
3. Tap notification.
4. Observe app navigation and message fetch.

Expected result:

- If implemented: app opens correct chat or message list and fetches.
- If not implemented: limitation is documented and not claimed complete.

Evidence:

- Screen recording or screenshots.
- Logs.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test PN11: Invalid Token Cleanup

Status: Pending if not implemented.

Purpose: Prevent stale token buildup and repeated FCM errors.

Steps:

1. Insert/register invalid token.
2. Send message to user.
3. Capture FCM response.
4. Query `device_tokens`.

Expected result:

- If implemented: invalid token is removed or marked invalid.
- If not implemented: limitation is documented and tracked.

Evidence:

- FCM response.
- SQL before/after.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```

## Test PN12: Push Disabled Server Mode

Purpose: Verify no-op notifier behavior is safe.

Steps:

1. Unset `GOOGLE_APPLICATION_CREDENTIALS`.
2. Start server.
3. Register device token.
4. Send message.

Expected result:

- Server logs no-op notifier.
- Message send still succeeds.
- No plaintext logs.
- No panic.

Evidence:

- Server logs.
- Message send response.

PASS/FAIL:

```text
Result:
Evidence path:
Notes:
```
