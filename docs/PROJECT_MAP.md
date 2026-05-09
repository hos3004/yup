# PROJECT_MAP — E2EE Secure Messaging (YUP)

> **Status:** Internal Alpha - M9/M10 stabilized
> **Last updated:** 2026-05-10 (M9/M10 final verification)
> **Architecture style:** Olm E2EE via Vodozemac (verified), Matrix-style session management

---

## [TECH_STACK]

| Layer | Technology | Version | Source | Status |
|-------|-----------|---------|--------|--------|
| **Mobile Framework** | Flutter | 3.35.7 | [docs.flutter.dev](https://docs.flutter.dev) | ✅ Verified |
| **Language (Mobile)** | Dart | 3.9.2 | [dart.dev](https://dart.dev) | ✅ Verified |
| **Rust (Crypto FFI)** | Rust toolchain | 1.95.0 (stable-gnu) | [blog.rust-lang.org](https://blog.rust-lang.org) | ✅ At target |
| **Crypto Library** | Vodozemac (Olm) | 0.10.0 | [crates.io](https://crates.io/crates/vodozemac) | ✅ Verified |
| **Backend** | Go | 1.26.2 | [go.dev](https://go.dev) | ✅ Verified |
| **Database** | PostgreSQL | 17 | [postgresql.org](https://www.postgresql.org) | ✅ Integrated via PostgresStore (runs in Docker) |
| **Realtime / Queue** | Redis | — | — | ⏳ **Not yet implemented** |
| **Local Storage** | SQLCipher | 4.15.0 | [github.com/sqlcipher](https://github.com/sqlcipher/sqlcipher) | ✅ Schema created and used |
| **Secure Key Storage** | Android Keystore / iOS Keychain | — | platform-native | ✅ Implemented via flutter_secure_storage |
| **Navigation** | GoRouter | 17.2.3 | [pub.dev](https://pub.dev/packages/go_router) | ✅ Wired |
| **Containerization** | Docker | 29.3.1 | [docs.docker.com](https://docs.docker.com) | ⏳ Needs upgrade to 29.4.1 |
| **CI/CD** | GitHub Actions | latest | [github.com](https://github.com) | ✅ Available |

---

## [ARCHITECTURE]

### Mobile App Architecture

```
mobile/lib/
├── main.dart
├── app/
│   ├── app.dart                  # MaterialApp + GoRouter setup
│   └── router.dart               # GoRouter route definitions
├── features/
│   ├── auth/
│   │   ├── data/                 # AuthRepository (API + storage)
│   │   ├── domain/               # AuthService (auth business logic)
│   │   └── presentation/         # RegisterScreen
│   ├── key_management/
│   │   ├── data/                 # DeviceRegistrationService, KeyRepository
│   │   ├── domain/               # CryptoService (Rust FFI wrapper)
│   │   └── presentation/         # (empty — no UI, background service)
│   ├── messaging/
│   │   ├── data/                 # SessionStore, PeerKeyStore
│   │   ├── domain/               # ConversationService (1:1)
│   │   └── presentation/         # ChatScreen
│   ├── verification/
│   │   ├── data/                 # VerificationService
│   │   ├── domain/               # (empty)
│   │   └── presentation/         # VerificationScreen (conversation fingerprint)
│   └── settings/
│       └── presentation/         # SettingsScreen (clear data deletes DB + passphrase)
├── core/
│   ├── networking/               # ApiClient (REST, auth-bearer)
│   ├── crypto_ffi/              # CryptoBridge (dart:ffi manual bindings)
│   ├── secure_storage/          # SecureStorageService wrapper
│   ├── storage/                  # LocalDatabase (SQLCipher), MessageDao
│   └── logging/                  # LogService (safe logging with key/token redaction)
└── shared/
    ├── widgets/                  # (empty)
    └── models/                   # (empty)
```

### Backend Architecture

```
server/
├── cmd/
│   └── main.go                   # Entry point with rate-limited routes, selects store by DATABASE_URL
├── internal/
│   ├── handler/                  # HTTP handlers + auth middleware (uses DataStore interface)
│   ├── middleware/               # Rate limiting (IP and user-based)
│   ├── service/
│   │   ├── store.go              # DataStore interface + InMemoryStore (fallback/tests)
│   │   └── postgres_store.go     # PostgresStore — production PostgreSQL-backed implementation
│   ├── model/                    # Domain models (User, Device, KeyBundle, Envelope)
├── migrations/                   # SQL migration files (golang-migrate compatible)
├── docker-compose.yml            # PostgreSQL 17 + server
├── Dockerfile                    # Multi-stage production build
├── Makefile                      # db-up/down/migrate/run/test targets
├── go.mod

### Crypto Architecture

```
Private Keys ─── NEVER leave the device

Library: Vodozemac 0.10 (Olm verified)
    └── C FFI layer in yup_crypto Rust crate
    └── Olm: 1:1 double-ratchet sessions (account, outbound/inbound, pickling)
    └── Fingerprint: SHA-256 of both identity keys in canonical sorted order

Communication with Flutter: manual dart:ffi binds
Key Storage: flutter_secure_storage (Keystore/Keychain-backed)
Protocols: Olm session establishment, SAS verification
Groups/Megolm: Deferred. No active M11 source code is part of M9/M10.
```

---

## [ADRs]

| ID | Title | Status |
|----|-------|--------|
| ADR-001–ADR-010 | (See docs/adr/) | ✅ All Approved |

---

## [KNOWN GAPS]

### Key Changed Warning — Detection Implemented (UI Pending Full Integration)
- **Issue:** Previously the app silently accepted key changes.
- **Fix (M7):** `PeerKeyStore` pins identity keys, detects changes, shows warning dialog.
- **Status:** Detection and warning UI coded. Requires end-to-end smoke test with real key rotation.

### Rust Host Build — MSVC (BLOCKER for Windows host release build)
- **Issue:** `msvcrt.lib` missing from VS 2022 Community installation.
- **Workaround:** Use `cargo +stable-gnu` for all Rust builds (tests and Android cross-compilation).
- **Android .so files:** Built successfully for x86_64 and arm64-v8a.
- **Rust toolchain:** 1.95.0 (at target).

### Inbound Session Persistence
- **Fix (M7):** `rust_create_inbound_session` now returns `{session_id, plaintext}`.
- Dart stores the actual inbound session ID for reply.
- Requires full app restart test to validate.

### QR Scanning (Pending)
- Verification is currently out-of-band text comparison only.

### M10 — Push Notifications
- `internal/notifier/notifier.go`: FCM push via Firebase Admin SDK, noop fallback.
- `POST /api/v1/devices`: Device token registration (auth-protected).
- Push sent async after `StoreMessage`.
- Flutter `PushService` provides `pushTriggers` stream for immediate polling.

### M11 — Group Chats
- Deferred and out of M9/M10 scope.
- No active group routes, GroupService, or Megolm app/server/Rust source code is present in the M9/M10 stabilization branch.
- Local developer PostgreSQL volumes may still contain old experimental group tables; those are not active source code.

---

## [MILESTONE STATUS]

| Milestone | Title | Status |
|-----------|-------|--------|
| M0 | Planning & Verification | ✅ Complete |
| M1 | Crypto Spike | ✅ Complete |
| M2 | Auth & Key Registry | ⚠️ Partially Done |
| M3 | 1:1 Secure Messaging | ⚠️ Partially Done |
| M4 | Key Verification | ⚠️ Partially Done |
| M5 | Local Secure Storage | ⚠️ Partially Done |
| M6 | MVP Hardening | ❌ Mostly Not Done |
| M6-FIX | Project Consistency & Readiness Cleanup | ✅ Completed |
| **M7** | **Security Correctness & Auth Hardening** | **✅ Completed** |
| **M7-FIX** | **Reaudit Blocking Fixes (6 blockers)** | **✅ All 6 resolved** |
| M8 | PostgreSQL Persistence | ✅ Completed |
| M9 | Security Verification & Evidence Pack | ✅ Internal Alpha stabilized |
| **M10** | **FCM Push Notifications** | **✅ Internal Alpha stabilized** |
| **M11** | **E2EE Group Chats (Megolm)** | **❌ Deferred / not active** |
