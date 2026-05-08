# ADR-005: Crypto Library

**Status:** ✅ Approved (Vodozemac 0.10 selected after spike)
**Date:** 2026-05-08

## Context
The MVP needs an E2EE cryptographic library. Candidates: matrix-rust-sdk-crypto (wraps Vodozemac with high-level session management), Vodozemac raw (direct Olm/Megolm bindings), and libsignal-client (Signal Protocol).

## Decision
**Vodozemac 0.10 raw as primary (matrix-rust-sdk-crypto rejected as fallback).**

- Vodozemac 0.10 selected after crypto spike validated it supports all required operations (Ed25519 key generation, Curve25519 O-T keys, Olm session establishment, encrypt/decrypt)
- Rust C FFI layer written directly against Vodozemac (flutter_rust_bridge considered but deferred)

## Alternatives Considered

| Alternative | Reason for Rejection |
|---|---|
| **matrix-rust-sdk-crypto 0.17** | Too heavy for MVP — pulls in unnecessary dependencies (matrix client, HTTP, state store) and adds complexity beyond what is needed for a basic Olm session flow |
| **libsignal-client** | Signal Protocol / X3DH / Double Ratchet is not a project goal; the project targets the Olm/Matrix cryptographic ecosystem for protocol alignment |
| **flutter_rust_bridge codegen** | Deferred — manual `dart:ffi` bindings give full control over the FFI surface and avoid codegen complexity; may be revisited post-MVP if the FFI surface grows significantly |
| **Direct libsodium X3DH** | Would require implementing the Olm protocol from scratch; Vodozemac provides a battle-tested implementation |

## Consequences
- **Positive:** Minimal dependency chain, fast compilation, full control over FFI surface
- **Positive:** Smaller binary (933KB .so for Android arm64)
- **Negative:** Must manually implement session management that matrix-sdk-crypto would have provided
- **Negative:** More Rust code to maintain
- **Future:** If session management becomes burdensome, matrix-rust-sdk-crypto may be reconsidered post-MVP

## References
- CRYPTO_SPIKE_DECISION.md — full evaluation report
- Vodozemac 0.10: https://crates.io/crates/vodozemac
