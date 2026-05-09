# M10 — Push Notifications (FCM)

> **Date:** 2026-05-09
> **Project:** YUP E2EE Secure Messaging
> **Test Suite:** All server tests passing (handler + service)

## Summary

Push notification infrastructure implemented using Firebase Cloud Messaging (FCM). The server sends data-only push notifications when a message is stored, replacing the need for constant polling. The Flutter client registers its FCM token on startup and uses incoming pushes to trigger immediate message fetches, with adaptive polling as a fallback.

## Architecture

```
[Flutter Client]                    [Server]
     │                                  │
     │  1. Register FCM token           │
     │  POST /api/v1/devices ─────────> │
     │                                  │
     │  2. Alice sends message          │
     │  POST /api/v1/messages ────────> │
     │                                  │── StoreMessage(sender, recipient, ...)
     │                                  │── GetDeviceTokens(recipient)
     │                                  │── SendPush (async, data-only)
     │                                  │
     │  3. Push received (foreground)   │
     │  onMessage handler ──────────>   │
     │  Trigger pollIncoming()          │
     │  GET /api/v1/messages ────────>  │
     │                                  │
     │  4. Fallback: adaptive polling   │
     │  (3s-30s interval)               │
```

## Server Changes

### DataStore Interface (store.go:27-28)
- `RegisterDeviceToken(username, token, platform string) error`
- `GetDeviceTokens(username string) ([]string, error)`

### InMemoryStore
- New `deviceTokens` field: `map[string]map[string]*model.DeviceToken` (username → token → DeviceToken)
- `RegisterDeviceToken`: Upserts with platform and timestamps
- `GetDeviceTokens`: Returns all token strings for a user
- `DeleteAllUserData`: Cleans up device tokens on user deletion

### PostgresStore (postgres_store.go)
- Inline migration creates `device_tokens` table:
  - `id BIGSERIAL PK`, `username FK`, `token`, `platform`, `created_at`, `updated_at`
  - `UNIQUE(username, token)` constraint
  - Index on `username`
- `RegisterDeviceToken`: `INSERT ... ON CONFLICT (username, token) DO UPDATE`
- `GetDeviceTokens`: `SELECT token FROM device_tokens WHERE username = $1`

### Migration File
- `migrations/000003_push_notifications.up.sql`: Standalone SQL for external migration tools

### Notifier Package (notifier/notifier.go — new)
- `Notifier` interface with `SendPush(ctx, tokens, data) (int, error)`
- `fcmNotifier`: Firebase Admin SDK implementation using `messaging.SendEachForMulticast`
- `noopNotifier`: No-op fallback when `GOOGLE_APPLICATION_CREDENTIALS` not set
- Auto-detects FCM configuration via environment variable

### Handler (handler.go)
- `Server` struct gains `notifier.Notifier` field
- `SendMessage`: After successful `StoreMessage`, asynchronously looks up recipient's device tokens and sends a data-only push with `{"type": "new_message", "sender": "<username>"}`
- `RegisterDevice` handler in `device.go` (new): Auth-protected `POST /api/v1/devices`

### Route (cmd/main.go)
- `POST /api/v1/devices` — rate-limited, auth-protected

## Flutter/Android Changes

### Dependencies (pubspec.yaml)
- `firebase_core: ^3.12.0`
- `firebase_messaging: ^15.2.0`

### Gradle
- Project-level `android/build.gradle.kts`: Google Services plugin `4.4.2`
- App-level `android/app/build.gradle.kts`: `com.google.gms.google-services` plugin

### Android Manifest
- `POST_NOTIFICATIONS` permission for Android 13+

### PushService (lib/core/push/push_service.dart — new)
- Initializes FCM, requests permissions, gets/registers device token
- Listens for token refresh and re-registers
- Listens for foreground `onMessage` and emits `pushTriggers` stream on `"new_message"` type

### ConversationService
- Accepts optional `pushTriggers` stream parameter
- On push trigger: resets poll interval to minimum and immediately runs poll cycle
- Cleans up `_pushSubscription` on dispose

### Service Container & Router
- `AppServices` includes `PushService`
- Router passes `services.push` to `ChatScreen`

### App Entry Point (main.dart)
- `Firebase.initializeApp()` before app startup
- `services.push.initialize()` registers FCM token

## Prerequisites (Manual Setup)

| Requirement | Location |
|-------------|----------|
| Firebase project with Cloud Messaging enabled | [Firebase Console](https://console.firebase.google.com) |
| `google-services.json` | `yup_mobile/android/app/google-services.json` |
| `GOOGLE_APPLICATION_CREDENTIALS` env var | Server environment (service account JSON path) |

## Test Suite

119/119 tests passing (same as pre-M10 baseline) — no regressions.
