# ADR-004: Platform Selection

**Status:** ✅ Approved
**Date:** 2026-05-08

## Context
The app targets Turkey and the Arab world initially. Android dominates (85%+) and iOS is significant (~15%). Web was considered.

## Decision
**Flutter for Android + iOS only. Web is deferred.**

- Flutter enables unified codebase for both mobile platforms
- RTL support (Arabic) and Turkish locale are mature in Flutter
- Web deferred due to:
  - WebCrypto limitations for private key storage
  - XSS risk
  - Difficulty of running Rust (Vodozemac/matrix-sdk-crypto) in browser via WASM
  - Increased attack surface for a security product

## Consequences
- **Positive:** Single codebase, consistent behavior across platforms
- **Positive:** Full access to platform Keystore/Keychain via plugins
- **Negative:** No Web version for desktop users
- **Future:** Web may be reconsidered post-MVP if WASM and WebCrypto improve for E2EE use cases
