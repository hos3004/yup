# PROJECT_MAP — E2EE Secure Messaging (YUP)

> **Status:** Technical Prototype with working E2EE flow — NOT production-ready, NOT closed-beta-ready
> **Last updated:** 2026-05-08
> **Architecture style:** Olm E2EE via Vodozemac (verified), Matrix-style session management

---

## [TECH_STACK]

| Layer | Technology | Version | Source | Status |
|-------|-----------|---------|--------|--------|
| **Mobile Framework** | Flutter | 3.41.5 | [docs.flutter.dev](https://docs.flutter.dev) | ✅ Verified |
| **Language (Mobile)** | Dart | 3.11.5 | [dart.dev](https://dart.dev) | ✅ Verified |
| **Rust (Crypto FFI)** | Rust toolchain | 1.92.0 (stable-gnu) | [blog.rust-lang.org](https://blog.rust-lang.org) | ⚠️ Below target (target: 1.95.0) |
| **Crypto Library** | Vodozemac (Olm) | 0.10.0 | [crates.io](https://crates.io/crates/vodozemac) | ✅ Verified |
| **matrix-rust-sdk-crypto** | — | 0.17.0 | [crates.io](https://crates.io/crates/matrix-sdk-crypto) | ❌ **Rejected** — too heavy for MVP |
| **Backend** | Go | 1.26.2 | [go.dev](https://go.dev) | ✅ Verified |
| **Database** | PostgreSQL | — | — | ⏳ **Not yet implemented** — server uses in-memory storage |
| **Realtime / Queue** | Redis | — | — | ⏳ **Not yet implemented** |
| **Local Storage** | SQLCipher | 4.15.0 | [github.com/sqlcipher](https://github.com/sqlcipher/sqlcipher) | ⚠️ Schema created but minimal usage |
| **Secure Key Storage** | Android Keystore / iOS Keychain | — | platform-native | ✅ Implemented via flutter_secure_storage |
| **State Management** | flutter_riverpod | 3.3.1 | [pub.dev](https://pub.dev/packages/flutter_riverpod) | ✅ Listed in dependencies but not yet used |
| **Navigation** | GoRouter | 17.2.3 | [pub.dev](https://pub.dev/packages/go_router) | ⚠️ Listed in dependencies but not yet wired |
| **Containerization** | Docker | 29.3.1 | [docs.docker.com](https://docs.docker.com) | ⏳ Needs upgrade to 29.4.1 |
| **CI/CD** | GitHub Actions | latest | [github.com](https://github.com) | ✅ Available |

---

## [SYSTEM_FLOW]

### Registration Flow
```
[User] → [App] → [Server]
  1. User enters desired username
  2. App sends POST /api/v1/users { username }
  3. Server validates uniqueness, creates user record
  4. Server returns user_id + auth_token
  5. App stores auth_token in secure storage
```

### Key Generation Flow
```
[App] → [Rust FFI: Vodozemac / Olm]
  1. App calls Rust FFI to generate identity keypair (Ed25519)
  2. App calls Rust FFI to generate one-time keypairs (Curve25519) — default 50
  3. Private keys stored in Android Keystore / iOS Keychain (via flutter_secure_storage)
  4. Public keys held in memory for upload
```

### Key Upload Flow
```
[App] → [Server]
  1. App sends PUT /api/v1/keys
     Body: { curve_key, ed_key, one_time_keys[], signature }
  2. Server stores public keys associated with user + device
  3. Server does NOT have access to plaintext or private keys
```

### Start Conversation Flow (Olm Session)
```
[Client A] → [Server] → [Client B]
  1. A searches for B by username
  2. A requests B's public keys: GET /api/v1/keys/:username
  3. A calls Rust FFI to establish Olm session using B's public keys
  4. Olm session created locally on A (no server involvement in session key exchange)
```

### Send Message Flow
```
[Client A] → [Server] → [Client B]
  1. A encrypts message using Olm session → ciphertext + message_type
  2. A sends POST /api/v1/messages
     Body: { sender, recipient, ciphertext, message_type, sender_key }
  3. Server stores ciphertext with metadata:
     - sender, recipient, timestamp, message_size_approx, delivery_status
  4. B fetches message by polling GET /api/v1/messages/:username
  5. B decrypts locally using Olm session
  6. B sends delivery acknowledgment via POST /api/v1/messages/:id/ack
```

### Key Verification Flow
```
[Client A] ↔ [Client B]
  1. A displays fingerprint (SHA-256 of both identity keys, formatted as hex groups)
  2. B displays fingerprint on their device
  3. A and B compare the displayed fingerprints out-of-band (in person or via trusted channel)
  4. On match: both mark each other as verified locally
     (verification status stored only on device, not server)
```

### Key Changed Flow (⚠️ NOT YET IMPLEMENTED)
```
[Client A] ← [Server]
  1. Server returns updated keys for B (different identity_key)
  2. A should detect key mismatch
  3. A should show warning: "Security code changed for [username]"
  4. User must acknowledge before continuing conversation

  ⚠️ Blocker: This flow is documented but NOT yet implemented.
  The app does not currently detect or warn on key changes.
```

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
│   │   ├── data/                 # SessionStore, MessageDao
│   │   ├── domain/               # ConversationService (encrypt/send/poll/decrypt)
│   │   └── presentation/         # ChatScreen
│   ├── verification/
│   │   ├── data/                 # VerificationService
│   │   ├── domain/               # (empty)
│   │   └── presentation/         # VerificationScreen
│   └── settings/
│       └── presentation/         # SettingsScreen
├── core/
│   ├── networking/               # ApiClient (REST), AuthInterceptor
│   ├── crypto_ffi/              # CryptoBridge (dart:ffi manual bindings)
│   ├── secure_storage/          # SecureStorageService wrapper
│   ├── storage/                  # LocalDatabase (SQLCipher), MessageDao
│   └── logging/                  # LogService (safe logging with key redaction)
└── shared/
    ├── widgets/                  # (empty — no reusable widgets yet)
    └── models/                   # (empty — no shared models yet)
```

> **Rule:** `core/` and `shared/` directories must only contain code that is demonstrably reused across 2+ features. Do not pre-emptively abstract.

### Backend Architecture

```
server/
├── cmd/
│   └── main.go                   # Entry point
├── internal/
│   ├── handler/                  # HTTP handlers + auth middleware
│   ├── middleware/               # Rate limiting
│   ├── service/                  # In-memory store (⚠️ NOT persistent)
│   ├── model/                    # Domain models (User, Device, KeyBundle, Message)
├── go.mod
└── main.exe
```

> ⚠️ **Critical gap:** The server uses in-memory storage (`map[string]*...` with `sync.RWMutex`). All data is lost on restart. PostgreSQL is not yet integrated. This is a **blocker** for any closed beta deployment.

### Crypto Architecture

```
┌──────────────────────────────────────────────────────┐
│                   Crypto Boundary                     │
│                                                       │
│  Private Keys  ─── NEVER leave the device              │
│                                                       │
│  Library: Vodozemac 0.10 (Olm verified)               │
│          └── C FFI layer in yup_crypto Rust crate     │
│                                                       │
│  Communication with Flutter: manual dart:ffi binds    │
│                                                       │
│  Key Storage:                                         │
│    • Android: flutter_secure_storage (Keystore-backed) │
│    • iOS: flutter_secure_storage (Keychain-backed)     │
│                                                       │
│  Protocols:                                           │
│    • Olm session establishment (Matrix-style)          │
│    • SAS verification (key fingerprint comparison)     │
│                                                       │
│  NOT used (MVP):                                      │
│    ❌ Signal Protocol / X3DH                           │
│    ❌ libsignal-client                                 │
│    ❌ Manual libsodium X3DH/Double Ratchet             │
│    ❌ Megolm (deferred — needed for groups)            │
└──────────────────────────────────────────────────────┘
```

### Storage Boundary

| Data | Storage | Encryption |
|------|---------|------------|
| Private keys | flutter_secure_storage | Hardware-backed (TEE/SE) |
| Olm sessions | flutter_secure_storage (pickled) | Platform-encrypted |
| Decrypted messages (local) | SQLCipher (schema exists, minimal usage) | AES-256 encrypted at rest |
| Ciphertext on server | In-memory (⚠️ no persistence) | Server cannot decrypt |
| Usernames / hashes | In-memory (⚠️ no persistence) | N/A |

### Trust Boundary

```
[Keystore/Keychain] — Fully Trusted (hardware isolated)
     ↓
[App Process] — Partially Trusted (OS integrity dependent)
     ↓ (ciphertext + metadata only)
[Network] — Untrusted
     ↓
[Server] — Untrusted (cannot read content)
     ├── Can see: metadata (sender, recipient, timestamp, size)
     └── Cannot see: message plaintext, private keys, session keys
     ↓ (ciphertext + metadata only)
[Network] — Untrusted
     ↓
[App Process] — Partially Trusted
     ↓
[Keystore/Keychain] — Fully Trusted
```

### ADR Index

| ID | Title | Status |
|----|-------|--------|
| ADR-001 | Identity Model: Username + QR/Invite in Beta | ✅ Approved |
| ADR-002 | Backend: Go Monolith Modular (Centralized) | ✅ Approved |
| ADR-003 | Open Source: After MVP, repo prepared from day 0 | ✅ Approved |
| ADR-004 | Platforms: Flutter Android + iOS only, Web deferred | ✅ Approved |
| ADR-005 | Crypto Library: Vodozemac 0.10 primary, matrix-sdk-crypto rejected | ✅ Approved |
| ADR-006 | Groups & Media: Deferred past MVP | ✅ Approved |
| ADR-007 | Message TTL: 30 days for undelivered messages | ✅ Approved (not yet implemented) |
| ADR-008 | Multi-device: Out of MVP, per-device keys | ✅ Approved |
| ADR-009 | Sealed Sender: Out of MVP, metadata visible | ✅ Approved |
| ADR-010 | Phone Discovery: Optional post-MVP with strict controls | ✅ Approved |

---

## [KNOWN GAPS]

### Server Persistence (BLOCKER for Closed Beta)
- **Issue:** Server uses in-memory maps (`service/store.go`). All users, keys, and messages are lost on restart.
- **Impact:** Cannot run any closed beta — server restart destroys all state.
- **Required:** PostgreSQL integration with SQL migrations before any beta deployment.
- **Plan:** Propose as a standalone milestone (M7: Server Persistence) after M6-FIX.

### Key Changed Warning (BLOCKER for security)
- **Issue:** If a user's keys change (device reset, account compromise), the app does not detect or warn. It silently uses the new keys.
- **Impact:** Users cannot detect man-in-the-middle key replacement.
- **Fix required before closed beta:** Detect identity key mismatch on session establishment and show a warning screen.

### QR Scanning (Pending — not in MVP)
- **Issue:** Verification is currently out-of-band text comparison only. QR scanning is not implemented.
- **Status:** Acceptable for technical prototype; QR recommended before closed beta.

### Navigation / Routing (Partially Done)
- **Issue:** `go_router` is listed in `pubspec.yaml` but was not wired. Navigation used raw `Navigator.pushReplacement`.
- **Fix (M6-FIX):** GoRouter now wired with registered routes. See `lib/app/router.dart`.

### Auth Layer Organization (Partially Done)
- **Issue:** `DeviceRegistrationService` in `key_management/data` handled both key management and auth logic.
- **Fix (M6-FIX):** `AuthRepository` + `AuthService` created in `features/auth/`. Key generation/upload remains in `key_management`.

### Testing (Mostly Missing)
- **Issue:** Only 1 widget test exists. No unit tests, no Go tests, no Rust tests.
- **Minimal set added (M6-FIX):** username validation, LogService redaction, Go handler validation.
- **Full test suite:** Deferred to M6.

### Settings Screen (Missing)
- **Issue:** No settings screen existed in the app.
- **Fix (M6-FIX):** Minimal settings screen created with username display, fingerprint, clear data, logout.

### Rust Toolchain Version
- **Issue:** Local Rust toolchain is 1.92.0, target is 1.95.0.
- **Impact:** May miss newer stdlib features; should upgrade before production builds.

---

## [ORPHANS & PENDING]

| Item | Status | Notes |
|------|--------|-------|
| Server persistence (PostgreSQL) | ❌ Blocker | In-memory only; ALL data lost on restart |
| Key changed warning | ❌ Blocker | Security-critical — user cannot detect key replacement |
| Go version verification | ✅ Verified | Go 1.26.2 confirmed |
| Redis version verification | ⏳ Pending | Redis 8.6.3 identified via web; not yet integrated |
| Docker version upgrade | ⏳ Pending | Local: 29.3.1, target: 29.4.1 |
| Rust version upgrade | ⏳ Pending | Local: 1.92.0, target: 1.95.0 |
| KVKK / BTK legal compliance | ⏳ Pending | Requires lawyer review; design supports data residency |
| App Store / Google Play compliance | ⏳ Pending | Review required before beta release |
| Phone discovery policy | ⏳ Pending | Post-MVP with rate limiting + abuse detection |
| Private Contact Discovery evaluation | ⏳ Pending | Post-MVP research item |
| Multi-device strategy | ⏳ Pending | Post-MVP; per-device keys, no shared private keys |
| Encrypted backup strategy | ⏳ Pending | Post-MVP; must not expose private keys |
| Media encryption strategy | ⏳ Pending | Post-MVP; ciphertext-only upload |
| Sealed Sender evaluation | ⏳ Pending | Post-MVP; reduces metadata leakage |
| Security audit vendor selection | ⏳ Pending | Required before public release |
| Reproducible builds | ⏳ Pending | Required for Open Source credibility |
| Detailed threat model document | ⏳ Pending | Needs dedicated security review |
| Server database migration tooling | ⏳ Pending | Choose golang-migrate or similar |
| QR scanning for key verification | ⏳ Pending | Recommended before closed beta |
| Comprehensive test suite | ⏳ Pending | Only minimal tests added in M6-FIX |

---

## [Milestone Progress]

| Milestone | Title | Status | Target |
|-----------|-------|--------|--------|
| M0 | Planning & Verification | ✅ Complete | Week 1 |
| M1 | Crypto Spike | ✅ **Complete** | Week 1 |
| M2 | Auth & Key Registry | ⚠️ Partially Done | Next (original) |
| M3 | 1:1 Secure Messaging | ⚠️ Partially Done | After M2 |
| M4 | Key Verification | ⚠️ Partially Done | After M3 |
| M5 | Local Secure Storage | ⚠️ Partially Done | After M4 |
| M6 | MVP Hardening | ❌ Mostly Not Done | After M5 |
| M6-FIX | Project Consistency & Readiness Cleanup | 🔵 Completed | Current |
| M7 | Server Persistence | ⏳ Not Started | Proposed |

### M1 Deliverables
- [x] Rust crate (`rust/`) with Vodozemac 0.10 Olm integration
- [x] C FFI exports for Flutter consumption
- [x] Cross-compilation for Android aarch64 (933KB .so)
- [x] Go server (`server/`) with message relay API (in-memory only)
- [x] Flutter project with feature-based structure
- [x] Dart FFI bridge (`crypto_bridge.dart`)
- [x] CryptoService abstraction
- [x] REST API client for Go server
- [x] Chat UI (registration, session setup, encrypt/send/decrypt)
- [x] CRYPTO_SPIKE_DECISION.md — Vodozemac selected
- [x] Verified: server stores ciphertext only, no plaintext

### M6-FIX Deliverables
- [x] PROJECT_MAP.md updated — honest milestone statuses, Known Gaps section added
- [x] ADR-005 through ADR-010 created with Context, Decision, Alternatives Considered, Consequences, Status
- [x] Server persistence gap documented as blocker
- [x] App architecture cleanup: `lib/app/app.dart` + `lib/app/router.dart` with GoRouter
- [x] Auth layer cleanup: `AuthRepository` in `features/auth/data`, `AuthService` in `features/auth/domain`
- [x] Verification gap documented (fingerprint only, no QR, no key changed warning)
- [x] Minimal `SettingsScreen` created
- [x] Basic tests added (username validation, LogService redaction, Go handler validation)
