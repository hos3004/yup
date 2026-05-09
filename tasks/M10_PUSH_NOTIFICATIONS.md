# M10 - Push Notifications (FCM)

Date: 2026-05-10

Status: Internal Alpha stabilized. This is not a Closed Beta readiness claim.

## Summary

Push notification infrastructure is implemented using Firebase Cloud Messaging (FCM). The server sends data-only push notifications when a message is stored. The Flutter client initializes Firebase at startup, but registers its FCM token with the YUP server only after registration/session restore has established auth.

## Architecture

```text
Flutter client                         Server
1. Firebase.initializeApp()
2. Register or restore session
3. PushService.initialize()
4. POST /api/v1/devices with bearer token
5. POST /api/v1/messages
6. StoreMessage(sender, recipient, ...)
7. GetDeviceTokens(recipient)
8. SendPush async, data-only payload
9. Foreground push triggers message fetch
```

## Server

- `DataStore.RegisterDeviceToken(username, token, platform string) error`
- `DataStore.GetDeviceTokens(username string) ([]string, error)`
- `InMemoryStore` stores tokens in memory and clears them on user data deletion.
- `PostgresStore` stores tokens in `device_tokens` with `UNIQUE(username, token)`.
- `POST /api/v1/devices` is auth-protected and rate-limited.
- `SendMessage` sends data-only push after message storage.
- No plaintext message content is included in the push payload.

## Flutter / Android

- `firebase_core` and `firebase_messaging` are included.
- Android `applicationId` is `yup.hossam.com`.
- `google-services.json` package name is `yup.hossam.com`.
- `Firebase.initializeApp()` runs in `main.dart`.
- `services.push.initialize()` runs from the registration/session-restore callback in `router.dart`.
- `PushService.initialize()` requests permission, gets the FCM token, registers it with the server, listens for token refresh, and exposes `pushTriggers`.

## Manual Prerequisites

| Requirement | Location |
| --- | --- |
| Firebase project with Cloud Messaging enabled | Firebase Console |
| `google-services.json` | `yup_mobile/android/app/google-services.json` |
| Service-account JSON | Server environment for real FCM delivery |
| `GOOGLE_APPLICATION_CREDENTIALS` | Server environment variable |

## Verification

Current command evidence is recorded in `docs/M9_M10_STABILIZATION_REPORT.md`.

Latest verified run:

- Go unit/vet/build: PASS.
- PostgreSQL integration with `DATABASE_URL_TEST`: PASS.
- `dart analyze lib/`: PASS.
- `flutter analyze`: PASS.
- `flutter test`: 56/56 PASS.
- Rust stable-gnu tests/builds: PASS.
- Android x64 and arm64 release APK builds: PASS.

## Remaining Before Closed Beta

- Real FCM delivery with service-account credentials must be smoke-tested.
- Foreground/background/resume notification behavior must be captured on device or emulator.
- Push payload capture must prove no plaintext, bearer token, private key, session key, pickle, or SQLCipher passphrase is present.
