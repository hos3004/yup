# M9/M10 Stabilization Report

Date: 2026-05-10

Branch: `m9-m10-success-fixes`

Status: Internal Alpha stabilized for M9/M10 verification. This is not a Closed Beta readiness claim.

## Summary

The M9/M10 blocker from the independent audit was fixed: PostgreSQL databases that still had the legacy `users.auth_token NOT NULL` column now migrate on startup to `token_hash`-only storage. The documented PostgreSQL integration command now passes against the existing Docker Compose database that previously failed.

## Changes

- `server/internal/service/postgres_store.go`
  - Auto-migration now drops legacy `users.auth_token` after ensuring `users.token_hash` exists.
- `server/migrations/000002_semantics_hardening.up.sql`
  - Message foreign key migration now drops existing constraints before adding them, making the migration safer to re-run on partially hardened databases.
- `server/migrations/000004_drop_auth_token.up.sql`
  - Migration now ensures `token_hash` exists before dropping `auth_token`.
- `yup_mobile/test/conversation_service_test.dart`
  - Removed analyzer-only test lint issues.
- `yup_mobile/test/fingerprint_bridge_test.dart`
  - Removed analyzer-only unused variable.

## Verification Commands

All commands below were run on 2026-05-10 from this branch.

### Git

Command:

```bash
git status --short --branch
```

Result before commit:

```text
## m9-m10-success-fixes
 M server/internal/service/postgres_store.go
 M server/migrations/000002_semantics_hardening.up.sql
 M server/migrations/000004_drop_auth_token.up.sql
 M yup_mobile/test/conversation_service_test.dart
 M yup_mobile/test/fingerprint_bridge_test.dart
```

### PostgreSQL Integration

Command:

```powershell
cd server
docker compose up -d postgres
$env:DATABASE_URL_TEST='postgres://yup:yup_dev@localhost:5432/yup?sslmode=disable'
& 'C:\Program Files\Go\bin\go.exe' test ./... -v -count=1 -p 1
```

Result:

```text
PASS
ok  	github.com/yup/server/internal/handler	3.267s
PASS
ok  	github.com/yup/server/internal/service	21.800s
```

Schema evidence after startup migration:

```text
Table "public.users"
username
display_name
created_at
token_hash
```

`auth_token` is no longer present in the PostgreSQL `users` table.

### Go Unit, Vet, Build

Command:

```powershell
cd server
& 'C:\Program Files\Go\bin\go.exe' test ./... -v -count=1
& 'C:\Program Files\Go\bin\go.exe' vet ./...
& 'C:\Program Files\Go\bin\go.exe' build ./cmd/main.go
```

Result:

```text
PASS
ok  	github.com/yup/server/internal/handler	1.046s
PASS
ok  	github.com/yup/server/internal/service	0.470s
```

`go vet ./...` and `go build ./cmd/main.go` completed with exit code 0.

### Flutter Analyze And Tests

Command:

```bash
cd yup_mobile
flutter pub get
dart analyze lib/
flutter analyze
flutter test
```

Result:

```text
Analyzing lib...
No issues found!
Analyzing yup_mobile...
No issues found!
00:01 +56: All tests passed!
```

### Android Builds

Command:

```bash
cd yup_mobile
flutter build apk --release --target-platform android-x64
flutter build apk --release --target-platform android-arm64
```

Result:

```text
Built build\app\outputs\flutter-apk\app-release.apk (34.0MB)
Built build\app\outputs\flutter-apk\app-release.apk (32.9MB)
```

### Rust

Command:

```bash
cd yup_mobile/rust
cargo +stable-gnu test
cargo +stable-gnu build --release
cargo +stable-gnu build --release --target x86_64-linux-android
cargo +stable-gnu build --release --target aarch64-linux-android
```

Result:

```text
test result: ok. 5 passed; 0 failed
Finished `release` profile
Finished `release` profile
Finished `release` profile
```

## Firebase And Push

Firebase package configuration:

```text
applicationId = "yup.hossam.com"
google-services.json package_name = "yup.hossam.com"
```

Push server registration timing:

- `Firebase.initializeApp()` still runs at startup.
- `services.push.initialize()` runs inside the registration/session-restore callback.
- `PushService.initialize()` is the only path that calls `ApiClient.registerDeviceToken`.

This means device-token registration is deferred until after auth/session restore sets the API token.

## M11 / Group Code Check

Source scan scope:

```text
server/internal
yup_mobile/lib
yup_mobile/rust/src
```

Search terms included `Megolm`, `GroupService`, `group_`, `/groups`, `RoomKey`, `InboundGroup`, `OutboundGroup`, `sendGroup`, `groupId`, and `group_id`.

Result: no active M11/group source code found in the scanned app/server/Rust source paths.

Note: the local Docker Compose PostgreSQL volume still contained old group tables from previous experiments. Those are residual database state, not active source code in this branch.

## Remaining Status

M9/M10 is now acceptable as Internal Alpha stabilization evidence.

Remaining work before Closed Beta still includes full manual mobile smoke testing, two-device E2EE evidence, push delivery evidence with real FCM credentials, and release-process signoff.
