# CRYPTO_SPIKE_DECISION — Milestone 1 Report

**Date:** 2026-05-08
**Status:** ✅ Crypto Spike Complete — MOVING FORWARD WITH VODOZEMAC

---

## Candidate Evaluation

### Candidate 1: matrix-rust-sdk-crypto 0.17.0

| Aspect | Assessment |
|--------|-----------|
| Purpose | Full Matrix E2EE protocol layer (OlmMachine, device tracking, key management, room events) |
| API Surface | Very large — designed for Matrix protocol, not generic 1:1 messaging |
| Dependencies | Heavy — depends on ruma (Matrix types), tokio, sqlite |
| Flutter FFI Complexity | High — requires wrapping complex async API with state management |
| Verdict | **NOT SUITABLE for MVP** — over-engineered for simple 1:1 encrypted text messaging |

**Reasons for rejection:**
- `matrix-sdk-crypto` is designed for Matrix protocol clients. Its encrypt/decrypt APIs work on Matrix `RoomId` + event types, not simple plaintext.
- Requires a database store (SQLite) even for basic operations.
- The async nature (tokio) adds complexity to the FFI bridge.
- For a simple 1:1 encrypted messenger, the Matrix protocol layer is unnecessary.

### Candidate 2: Vodozemac 0.10.0 (Olm/Megolm)

| Aspect | Assessment |
|--------|-----------|
| Purpose | Pure Rust implementation of Olm and Megolm cryptographic ratchets |
| API Surface | Minimal — Account, Session, PreKeyMessage, OlmMessage |
| Dependencies | Light — curve25519-dalek, ed25519-dalek, sha2, AES |
| Flutter FFI Complexity | Low — synchronous API, easy C FFI wrapper |
| Cross-compilation | ✅ Builds for aarch64-linux-android, armv7, x86_64 |
| License | Apache 2.0 ✅ |
| Security Audit | ✅ Least Authority audit, no significant findings |
| Version | 0.10.0 (April 2026) — active development |
| Verdict | **✅ RECOMMENDED for MVP** |

### Candidate 3: libsignal-client (Signal Protocol)

| Aspect | Assessment |
|--------|-----------|
| Decision | ❌ **EXCLUDED from MVP** (per ADR-005) |
| Reason | Signal's library is tightly coupled to their server protocol and has AGPL license constraints |

### Candidate 4: Manual X3DH/Double Ratchet with libsodium

| Aspect | Assessment |
|--------|-----------|
| Decision | ❌ **FORBIDDEN** (per ADR-005) |
| Reason | Implementing cryptographic protocols manually is error-prone and high-risk for a security product |

---

## Proof of Concept Results

### What was built

1. **Rust crate** (`yup_crypto`) with:
   - Olm `Account` creation and key management
   - Ed25519 signing
   - Outbound/inbound Olm session creation
   - Message encryption/decryption via Olm ratchet
   - SHA-256 fingerprint computation
   - C FFI export layer for Flutter integration

2. **Go server** (`server/`) with:
   - User registration (username-based)
   - Key bundle upload and retrieval
   - Message relay (ciphertext only, no plaintext)
   - REST API (REST: POST/GET users, keys, messages)
   - Polling-based message delivery (WebSocket deferred to M2)

3. **Flutter app** (`yup_mobile/`) with:
   - Feature-based directory structure (auth, key_management, messaging, verification, settings)
   - C FFI bridge via `dart:ffi`
   - REST API client for Go server
   - Chat UI with registration, session setup, encrypt/send/decrypt flow
   - SQLCipher secure storage structure prepared (M5)

### Verified Capabilities

| # | Capability | Status |
|---|-----------|--------|
| 1 | Rust FFI compiles for Windows (GNU) | ✅ |
| 2 | Rust FFI cross-compiles for Android arm64 | ✅ |
| 3 | Go server builds and runs | ✅ |
| 4 | Key generation on device | ✅ |
| 5 | Outbound Olm session creation | ✅ |
| 6 | Inbound Olm session from pre-key message | ✅ |
| 7 | Message encryption | ✅ |
| 8 | Message decryption | ✅ |
| 9 | Server stores only ciphertext | ✅ |
| 10 | No plaintext in server database | ✅ |
| 11 | Private keys never leave device | ✅ (by design — FFI boundary) |

### What the Server Sees (Metadata Visibility)

| Data | Visible to Server? |
|------|-------------------|
| Message plaintext | ❌ No |
| Private keys | ❌ No |
| Ciphertext | ✅ Yes (cannot decrypt) |
| Sender username | ✅ Yes |
| Recipient username | ✅ Yes |
| Timestamp | ✅ Yes |
| Approximate message size | ✅ Yes |
| Delivery status | ✅ Yes |

---

## Final Decision

**✅ MOVE FORWARD WITH VODOZEMAC 0.10.0**

- matrix-rust-sdk-crypto is too heavy for the MVP's simple 1:1 messaging model
- Vodozemac provides exactly the cryptographic primitives needed (Olm)
- FFI bridge is straightforward (C-compatible exports)
- No async complexity in the crypto layer
- Apache 2.0 license compatible with project goals
- Future Megolm support for groups (post-MVP)

## Next Steps

1. ✅ M1 complete — proceed to M2 (Auth & Key Registry)
2. M2: Integrate Flutter UI with Go server auth + key upload
3. M3: Build full 1:1 messaging flow with send/receive
4. Update PROJECT_MAP.md to reflect Vodozemac as the crypto library
