# ADR-008: Multi-device

**Status:** ✅ Approved
**Date:** 2026-05-08

## Context
Multi-device support (same identity, multiple devices) adds significant complexity: device key management, session fan-out, out-of-order message handling, device history.

## Decision
**Multi-device is out of MVP scope. Per-device keys only.**

- Each device has its own independent identity keypair
- No mechanism to link multiple devices under one user identity in MVP
- When multi-device is added later, it will use per-device keys (no shared private keys)
- Session key export/import avoided as a design goal — never share private key material between devices

## Alternatives Considered

| Alternative | Reason for Rejection |
|---|---|
| **Signal-style master secret + device derived keys** | Requires a master secret that must be shared across devices — contradicts the "private keys never leave the device" principle |
| **Matrix-style device keys + key backup** | Viable but complex; requires encrypted key backup infrastructure and secure recovery mechanism; deferred as post-MVP |
| **Session key export/import** | Sharing session keys between devices weakens the E2EE model and introduces export/import UI complexity; rejected by design |
| **Include basic multi-device in MVP** | Would add device listing, session fan-out, and out-of-order message handling — significant scope increase for uncertain benefit at MVP stage |

## Consequences
- **Positive:** Simpler MVP — one device per account, no fan-out logic
- **Positive:** Stronger security — no mechanism for private keys to leave the device
- **Negative:** Users cannot resume conversations on a new device without re-establishing sessions
- **Future:** Multi-device added post-MVP using per-device key model; session recovery via secure backup

## References
- Matrix multi-device spec: https://spec.matrix.org/latest/client-server-api/#key-management
- ADR-010: Private Contact Discovery (related deferred feature)
