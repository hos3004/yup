# M9_M10_INDEPENDENT_AUDIT_REPORT.md

Date: 2026-05-09
Auditor role: Independent Staff-level Security & Code Auditor
Repository: `https://github.com/hos3004/yup`
Local checkout: `H:\projects\yup`
Audited state: local working tree, not a clean commit
HEAD: `3fb510076fe9d3fd4443d962fcbd59024ae6164d`
HEAD label: `3fb5100 M7-FIX: Resolve all 6 reaudit blocking fixes`
Remote heads observed: only `master` at `3fb510076fe9d3fd4443d962fcbd59024ae6164d`

Verdict: **M9/M10 Rejected, critical fixes required**

Classification: **Internal Alpha with critical blockers**

The current local app is not a M9/M10-ready artifact. It is a dirty local working tree containing uncommitted M8.1, M10, and M11-era changes. The Android app does not build. The Rust crypto source does not compile. The full required PostgreSQL integration command fails unless Go package parallelism is manually disabled. The M9 evidence document contains claims contradicted by local command output. M10 push notification infrastructure exists on the server, but the Flutter integration is broken at startup and Android Firebase configuration is invalid.

---

## 1. Scope And Repository State

| Item | Status | Evidence |
|---|---:|---|
| Latest remote branch found | PASS | `git ls-remote --heads https://github.com/hos3004/yup.git` returned only `refs/heads/master` at `3fb5100...`. |
| Tags found | PASS | `git ls-remote --tags` returned no tags. |
| Local checkout matches remote HEAD | PASS | `git rev-parse HEAD` and `git rev-parse origin/master` both returned `3fb510076fe9d3fd4443d962fcbd59024ae6164d`. |
| Local tree clean | FAIL | `git status --short --branch` shows many modified and untracked files, including M10/M11 files. This is not an auditable release commit. |
| M10/M11 changes committed | FAIL | M10 files such as `tasks/M10_PUSH_NOTIFICATIONS.md`, `server/internal/handler/device.go`, `server/internal/notifier/`, `yup_mobile/lib/core/push/`, and `yup_mobile/android/app/google-services.json` are untracked or modified, not committed. |

Observed `git log --oneline -8`:

```text
3fb5100 M7-FIX: Resolve all 6 reaudit blocking fixes
d58b3c0 M9: Security Verification & Evidence Pack
96ec243 M8: PostgreSQL Persistence Foundation
6b6185f M7: Security Correctness & Server Auth Hardening
90f7e9f Initial commit: Yup E2EE messaging app
```

Important: this audit targets the **local working tree** because the user requested local review of M9/M10 and the M10 implementation is not available as a clean branch or commit.

---

## 2. Commands Run

| Command | Status | Evidence |
|---|---:|---|
| `git status --short --branch` | FAIL | Dirty tree. Modified M9 docs, server, Flutter, Rust, Android files; untracked M10/M11 files. |
| `git log --oneline -8` | PASS | Latest committed M9 evidence commit is `d58b3c0`; current HEAD is `3fb5100`. |
| `flutter doctor -v` | PASS | Flutter 3.35.7, Dart 3.9.2, Android SDK 36.1.0, Android emulator present; no issues found. |
| `flutter devices` | PASS | 4 devices found, including `sdk gphone64 x86 64` / `emulator-5556` / `android-x64`. |
| `cd yup_mobile && flutter pub get` | PASS | Dependencies resolved; 24 packages had newer incompatible versions. |
| `cd yup_mobile && dart analyze lib/` | PARTIAL | Exit 0, but 2 info issues: `prefer_initializing_formals` in `conversation_service.dart:112`, `use_build_context_synchronously` in `chat_screen.dart:90`. |
| `cd yup_mobile && flutter analyze` | FAIL | Exit 1 with 6 issues, including warnings in tests: unused import, unused field, unused local variable. |
| `cd yup_mobile && flutter test` | PASS | 56 Flutter/Dart tests passed. |
| `cd server && go test ./... -v -count=1` | PASS | Server tests passed when `DATABASE_URL_TEST` was not set; Postgres integration tests were skipped. |
| `cd server && go vet ./...` | PASS | Exit 0, no output. |
| `cd server && go build ./cmd/main.go` | PASS | Exit 0. |
| `cd server && docker compose up -d postgres` | PASS | `server-postgres-1` running and healthy on port 5432. |
| `DATABASE_URL_TEST=... go test ./... -v -count=1` | FAIL | Required command fails reproducibly under default package parallelism. First run failed `TestStoreSuite_Postgres/Postgres/MessageLifecycle` with `fetch: got 401`; rerun failed `TestStoreSuite_Postgres/Postgres/GetSentMessages` with `send 1: got 401`. |
| `DATABASE_URL_TEST=... go test ./... -v -count=1 -p 1` | PASS | Serial package execution passes. This proves the default required command is not isolated and packages clobber the same Postgres DB in parallel. |
| `cd server && docker compose build server` | PASS | Docker image `server-server:latest` built successfully. |
| `make --version` | BLOCKED | `make` is not installed in this Windows audit environment. |
| `cd yup_mobile/rust && cargo test` | FAIL | MSVC host build failed with `LNK1104: cannot open file 'msvcrt.lib'`. |
| `cd yup_mobile/rust && cargo build --release` | FAIL | Same MSVC `msvcrt.lib` linker failure. |
| `cd yup_mobile/rust && cargo +stable-gnu test` | FAIL | Rust source compile errors: `OutboundGroupSession` not found, ambiguous/wrong `SessionConfig`, `InboundGroupSession::new` wrong args, `decrypt` return type mismatch. |
| `cargo +stable-gnu build --release --target x86_64-linux-android` | FAIL | Same Rust source compile errors. |
| `cargo +stable-gnu build --release --target aarch64-linux-android` | FAIL | Same Rust source compile errors. |
| `flutter build apk --debug --target-platform android-x64` | FAIL | `:app:processDebugGoogleServices` failed: no matching client for package `com.yup.yup_mobile` in `google-services.json`. |
| `flutter build apk --debug --target-platform android-arm64` | FAIL | Same Firebase package mismatch. |
| Local app run on emulator | BLOCKED | Android build fails before install/run. |

---

## 3. Build And Local App Verdict

| Check | Status | Evidence |
|---|---:|---|
| Android emulator architecture has native library | PASS | Emulator is `android-x64`; `yup_mobile/android/app/src/main/jniLibs/x86_64/libyup_crypto.so` exists. |
| `arm64-v8a` native library exists | PASS | `yup_mobile/android/app/src/main/jniLibs/arm64-v8a/libyup_crypto.so` exists. |
| Rust source can reproduce `.so` artifacts | FAIL | Current `yup_mobile/rust/src/lib.rs` does not compile under `stable-gnu` for Android targets. Existing `.so` files are stale artifacts relative to source. |
| Android app builds | FAIL | Both x86_64 and arm64 debug APK builds fail due invalid Firebase `google-services.json`. |
| App can be smoke-tested locally | FAIL | Emulator exists, but app cannot be installed because APK build fails. |

Android build failure:

```text
Execution failed for task ':app:processDebugGoogleServices'.
No matching client found for package name 'com.yup.yup_mobile'
in H:\projects\yup\yup_mobile\android\app\google-services.json
```

The mismatch is direct:

- `yup_mobile/android/app/build.gradle.kts`: `applicationId = "com.yup.yup_mobile"`
- `yup_mobile/android/app/google-services.json`: `package_name = "yup.hossam.com"`

This alone rejects M10 as a runnable local app milestone.

---

## 4. M9 Evidence Pack Verification

| M9 claim | Status | Evidence |
|---|---:|---|
| "99/99 tests passing" | FAIL | Current local `flutter test` reports 56 tests, not 36. Rust tests fail. Full Postgres command fails unless `-p 1` is added. |
| "Go Server Tests 58/58 PASS" | PARTIAL | Non-DB Go tests pass. Required `DATABASE_URL_TEST=... go test ./...` fails under default Go package parallelism. |
| "Flutter/Dart Tests 36/36 PASS" | STALE | Current local output is 56/56 passing, not 36/36. The document is not accurate for the audited tree. |
| "Rust Crypto Tests 5/5 PASS" | FAIL | `cargo test` fails under MSVC; `cargo +stable-gnu test` also fails due source compile errors in Megolm additions. |
| "`dart analyze lib/` no issues" | FAIL | `dart analyze lib/` exits 0 but reports 2 info issues. |
| "`flutter analyze` no issues" | FAIL | `flutter analyze` exits 1 with 6 issues. |
| Token hashed at rest in PostgreSQL | FAIL | `postgres_store.go:192-194` inserts both `auth_token` and `token_hash`; DB query showed `auth_token <> ''` true for test users. |
| Sender key derived from registered key | PASS | `server/internal/handler/handler.go:205-211` derives sender key from `GetCurveKey(sender)`. Manual smoke showed response `sender_curve_key = YUtleQ==` for Alice. |
| Empty Postgres queues return `[]` | PASS | Manual Postgres smoke: authenticated fetch by non-recipient returned `[]`, not `null`. |
| M9 does not claim Closed Beta Ready | PASS | `docs/M9_SECURITY_VERIFICATION.md` lists blockers before Closed Beta/Public Beta. |

### Token At Rest Finding

M9 claims token hashing is implemented. It is not implemented cleanly. PostgreSQL stores both the plaintext token and the hash.

Source evidence:

- `server/internal/service/postgres_store.go:44-49` creates `auth_token` and `token_hash`.
- `server/internal/service/postgres_store.go:192-194` inserts `username, auth_token, token_hash`.
- `server/internal/service/postgres_store.go:216` still selects `auth_token`.

Database evidence from manual test:

```text
m9a1778353640338|t|t
m9b1778353640338|t|t
```

That query was:

```sql
SELECT username, auth_token <> '' AS has_auth_token, token_hash <> '' AS has_token_hash
FROM users
WHERE username IN ('m9a1778353640338','m9b1778353640338')
ORDER BY username;
```

Status: **FAIL**. A plaintext bearer token column remains populated.

---

## 5. M10 Push Notifications Verification

| M10 claim | Status | Evidence |
|---|---:|---|
| `POST /api/v1/devices` exists | PASS | `server/cmd/main.go:56`, `server/internal/handler/device.go:10-40`. |
| Device registration is auth-protected | PASS | Route is wrapped with `RateLimitAuth(AuthMiddleware(...))`; manual unauthenticated call returned 401. |
| Authenticated device registration works | PASS | Manual Postgres smoke returned `{"status":"registered"}`. |
| Server stores FCM device token | PASS with risk | DB query returned the exact token string in `device_tokens`. This works, but is plaintext token storage. |
| Server sends push after message store | PARTIAL | `handler.go:217-233` asynchronously calls notifier. No local FCM credential was configured, so runtime used no-op notifier. |
| Push payload excludes plaintext | PASS | Server sends `{"type":"new_message","sender":sender}` only. |
| Firebase Admin SDK fallback works | PARTIAL | No-op fallback logs when `GOOGLE_APPLICATION_CREDENTIALS` is missing. This prevents crashes but means local push delivery is not proven. |
| Flutter registers FCM token on startup | FAIL | `main.dart:8-10` initializes Firebase and calls `services.push.initialize()` before registration/session restore sets an auth token. If `getToken()` returns non-null, `PushService` calls authenticated `/devices` with no Bearer token and throws before `runApp`. |
| Flutter push supports background/resume | FAIL | Only `FirebaseMessaging.onMessage.listen` exists. No `onBackgroundMessage`, no `onMessageOpenedApp`, no initial message handling. |
| Android Firebase config valid | FAIL | `google-services.json` package is `yup.hossam.com`; app id is `com.yup.yup_mobile`; Android build fails. |
| M10 test coverage exists | FAIL | Search found no tests for `RegisterDevice`, `RegisterDeviceToken`, `GetDeviceTokens`, `SendPush`, `PushService`, or Firebase message handling. |
| Invalid FCM token cleanup | FAIL | `notifier.SendPush` returns success count only; failed token responses are not inspected and `device_tokens` are never pruned. |

Manual M10 server smoke result against PostgreSQL:

```json
{
  "health": "ok",
  "unauth_device_status": 401,
  "auth_device_status": "registered",
  "unauth_fetch_status": 401,
  "sent_message_id": "f05dacb323600c626b22cc9619689265",
  "sent_sender": "m9a1778353640338",
  "sent_sender_key": "YUtleQ==",
  "a_fetch_body": "[]",
  "b_fetch_count": 1,
  "wrong_ack_status": 400,
  "recipient_ack_status": "acknowledged",
  "fcm_token": "fcm_token_secret_1778353640338"
}
```

Plaintext device-token storage evidence:

```text
m9b1778353640338|fcm_token_secret_1778353640338|android
```

Status: M10 server skeleton is present; M10 app integration is **not acceptable**.

---

## 6. PostgreSQL Integration Test Harness

The required command:

```powershell
DATABASE_URL_TEST=postgres://yup:yup_dev@localhost:5432/yup?sslmode=disable go test ./... -v -count=1
```

fails under default Go package parallelism.

Observed failures:

```text
TestStoreSuite_Postgres/Postgres/MessageLifecycle
handler_store_test.go:164: fetch: got 401
```

Rerun:

```text
TestStoreSuite_Postgres/Postgres/GetSentMessages
handler_store_test.go:217: send 1: got 401
```

Then this passed:

```powershell
DATABASE_URL_TEST=... go test ./... -v -count=1 -p 1
```

Conclusion: Postgres integration tests share one database and are not isolated across packages. The default `go test ./...` command is not reliable when `DATABASE_URL_TEST` is set. This invalidates "all integration tests pass" as a strict CI-ready claim unless CI uses serial package execution or isolated databases/schemas.

---

## 7. Migrations And Makefile

| Check | Status | Evidence |
|---|---:|---|
| M8.1 migration file exists | PASS | `server/migrations/000002_semantics_hardening.up.sql`. |
| M10 migration file exists | PARTIAL | `server/migrations/000003_push_notifications.up.sql` exists. No matching `.down.sql` exists. |
| M8.1 migration SQL valid | FAIL | It uses `ALTER TABLE ... ADD CONSTRAINT IF NOT EXISTS`, which PostgreSQL rejects. Reproduced with `ERROR: syntax error at or near "NOT"`. |
| Runtime migration uses files | FAIL | `PostgresStore` uses embedded `migrationUpSQL`, not migration files. |
| Makefile can be run locally | BLOCKED | `make` not installed. |
| Makefile migration DSN matches compose | FAIL | Makefile uses `postgres://yup:yup_pass@postgres:5432/yupdb`; compose uses `postgres://yup:yup_dev@postgres:5432/yup`. |
| Makefile `test-integration` sets DB env | FAIL | Target does not set `DATABASE_URL_TEST`; it will skip Postgres tests unless caller supplies env. |

---

## 8. Local M11 Contamination / Security Regressions

Although the user requested M9/M10, the local tree includes M11 group-chat files and `PROJECT_MAP.md` claims M11 completed. These changes contaminate the local app safety posture and cannot be ignored.

| Finding | Status | Evidence |
|---|---:|---|
| Group metadata routes are not authenticated | FAIL | `cmd/main.go:61` registers `GET /api/v1/groups/{groupID}` as `h.RateLimitAuth(h.GetGroup)` and `cmd/main.go:64` registers members similarly. `RateLimitAuth` is only rate limiting; it does not authenticate. |
| Any authenticated user can add/remove group members | FAIL | `group.go:51-69` and `group.go:72-85` never verify caller is creator/admin/member. |
| Postgres group message send does not check membership | FAIL | `postgres_store.go:819-835` inserts group messages without verifying sender membership. InMemory does check membership at `store.go:498-503`, so stores are inconsistent. |
| Rust Megolm implementation compiles | FAIL | `cargo +stable-gnu test` fails due Vodozemac Megolm API/type errors. |
| Group session key export parsed correctly | FAIL | Rust `rust_export_group_session` returns JSON `{"session_key":...}`; Dart `CryptoBridge.exportGroupSession` uses `_parseResultString`, returning the raw JSON string as if it were a key. |

These M11 issues are outside M10 acceptance, but they prove the current local application is not safe to describe as security-hardened or beta-bound.

---

## 9. Documentation Vs Reality

| Document claim | Status | Evidence |
|---|---:|---|
| `docs/M9_SECURITY_VERIFICATION.md`: all 99 tests pass | FAIL | Current local Rust fails; Flutter analyze fails; Postgres required command fails. |
| `docs/M9_SECURITY_VERIFICATION.md`: `flutter analyze` no issues | FAIL | `flutter analyze` returned 6 issues and exit 1. |
| `docs/M9_SECURITY_VERIFICATION.md`: Rust tests pass with GNU workaround | FAIL | `cargo +stable-gnu test` fails with source compile errors. |
| `docs/M9_SECURITY_VERIFICATION.md`: token hashing at rest | FAIL | Plaintext `auth_token` remains populated in PostgreSQL. |
| `tasks/M10_PUSH_NOTIFICATIONS.md`: "119/119 tests passing" | FAIL | No command output supports this. Current local `flutter test` has 56 tests; M10 push has no dedicated tests. |
| `tasks/M10_PUSH_NOTIFICATIONS.md`: Flutter client registers FCM token on startup | FAIL | Code does so before authentication, so it cannot succeed reliably and can block `runApp`. |
| `PROJECT_MAP.md`: M10 completed | FAIL | Android build fails and push client startup is broken. |
| `PROJECT_MAP.md`: M11 completed | FAIL | Rust Megolm code does not compile; group routes have authorization defects. |
| `PROJECT_MAP.md`: Rust toolchain 1.95.0 | FAIL | Local `cargo --version` reported `cargo 1.92.0`; rustup active default is `stable-x86_64-pc-windows-msvc`. |
| No Closed Beta Ready claim | PASS | Docs still avoid direct "Closed Beta Ready" language. |

---

## 10. Security Logging

| Check | Status | Evidence |
|---|---:|---|
| Client LogService redacts common tokens/keys | PASS | `flutter test` includes LogService tests and passed. |
| Client logging tests inspect actual developer log output | PARTIAL | Tests mostly assert no throw or direct redaction helper output; they do not capture `developer.log`. |
| Server logs plaintext message bodies | PASS | No direct server logging of ciphertext/plaintext request bodies found in handlers. |
| Server logs push tokens | PARTIAL | No direct token logging in notifier. However FCM error strings are logged raw in `handler.go:229-230`; depending on Admin SDK error content, this may leak token metadata. |
| Device tokens stored encrypted/hashed | FAIL | Device tokens are stored plaintext in `device_tokens.token`. |

---

## 11. Final PASS / FAIL Matrix

| Area | Result |
|---|---:|
| Build environment | PARTIAL |
| Flutter dependency resolution | PASS |
| Dart lib analysis | PARTIAL |
| Full Flutter analysis | FAIL |
| Flutter tests | PASS |
| Go unit tests without Postgres env | PASS |
| Go vet | PASS |
| Go build | PASS |
| PostgreSQL integration required command | FAIL |
| PostgreSQL integration with `-p 1` | PASS |
| Docker Compose Postgres | PASS |
| Docker server image build | PASS |
| Rust tests | FAIL |
| Rust Android source builds | FAIL |
| Existing Android native libs present | PASS |
| Android APK x86_64 build | FAIL |
| Android APK arm64-v8a build | FAIL |
| Local emulator app smoke | BLOCKED |
| M9 evidence pack accuracy | FAIL |
| M10 server device registration | PARTIAL |
| M10 Flutter push integration | FAIL |
| M10 push test coverage | FAIL |
| Documentation honesty | FAIL |

---

## 12. Top Critical Issues

1. Android app does not build because `google-services.json` package name does not match `applicationId`.
2. Current Rust crypto source does not compile under `stable-gnu`; Megolm additions use wrong Vodozemac API/types.
3. Required PostgreSQL integration command fails under default `go test ./...` package parallelism.
4. PostgreSQL still stores plaintext `auth_token` even while also storing `token_hash`.
5. M10 `PushService.initialize()` runs before auth token setup and can throw before `runApp`.
6. M10 has no background/resume notification handling.
7. M10 has no dedicated tests for device registration, FCM notifier, token refresh, or push-triggered polling.
8. Existing `.so` libraries are stale artifacts; current Rust source cannot reproduce them.
9. Migration file `000002_semantics_hardening.up.sql` contains invalid PostgreSQL syntax.
10. Local M11 group routes introduce severe authorization defects and should not be present in a M9/M10 acceptance tree.

## 13. Medium Issues

1. `flutter analyze` fails due warnings in tests.
2. `dart analyze lib/` reports info issues despite doc claiming no issues.
3. `Makefile` integration target does not set `DATABASE_URL_TEST`.
4. `Makefile` migration DSN does not match `docker-compose.yml`.
5. No down migration exists for `000003_push_notifications`.
6. FCM failed-token cleanup is not implemented.
7. Device token platform validation is minimal and hard-coded client side to Android.
8. Server FCM notifier falls back to no-op silently enough that local push delivery can be claimed without proof.
9. Test counts in docs are stale/false for the audited tree.
10. Project map claims M10/M11 completed despite build/test failures.

## 14. Low Issues

1. `rg` could not be used locally due `Access is denied`; audit used PowerShell alternatives.
2. Docs contain mojibake/encoding artifacts in several rendered tables/arrows.
3. `PROJECT_MAP.md` lists Docker needing 29.4.1 while local Docker is 29.3.1; not tied to a demonstrated failure.
4. `PushService.dispose` is not visibly called from the app-level service container lifecycle.
5. `ApiClient.sendMessage` still accepts/sends `sender_key` even though server ignores it.
6. `LogService` tests do not capture real `developer.log` output.
7. `GetDeviceTokens` returns nil slice in Postgres when no rows; not directly API-encoded today, but inconsistent with strict empty-list semantics.
8. `RegisterDevice` returns raw internal DB errors to client on failure.
9. M10 task is in `tasks/`, not a formal evidence report under `docs/`.
10. Dirty working tree makes reproducibility poor.

---

## 15. Blockers Before Closed Beta

- Produce a clean M9/M10 commit or branch. Do not audit dirty uncommitted feature soup.
- Fix Android Firebase configuration and prove APK builds for emulator x86_64 and arm64-v8a.
- Fix Rust source so `cargo +stable-gnu test` and Android target builds pass from source.
- Remove plaintext `auth_token` storage from PostgreSQL or document and mitigate migration from legacy tokens.
- Fix Postgres integration test isolation so the exact required `go test ./...` command passes with `DATABASE_URL_TEST`.
- Move push registration to an authenticated point after login/session restore, and retry registration when auth becomes available.
- Add M10 tests: device registration auth, token upsert, invalid token cleanup, no-op notifier, FCM notifier with mock, push-triggered polling, token refresh.
- Add background/resume notification handling or document foreground-only scope honestly.
- Fix invalid migration SQL and add missing down migration for M10.
- Remove or quarantine M11 until group authorization and Megolm compilation are fixed.

## 16. Recommended Next Milestone

Recommended next milestone: **M9/M10 Stabilization and Reproducible Local App Build**

Acceptance criteria:

1. Clean branch/commit with no uncommitted M10/M11 work.
2. `flutter analyze`, `flutter test`, `go test ./...`, `go vet ./...`, `DATABASE_URL_TEST=... go test ./...`, `cargo +stable-gnu test`, and Android x86_64/arm64 builds all pass from a clean checkout.
3. Android app installs and registers two users locally.
4. Push token registration occurs only after auth and is tested.
5. PostgreSQL stores no plaintext auth tokens.
6. Docs are regenerated from real command output, not optimistic summaries.

Do not proceed to M11 or Closed Beta work from this tree.
