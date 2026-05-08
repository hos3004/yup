# ADR-003: Open Source Strategy

**Status:** ✅ Approved
**Date:** 2026-05-08

## Context
Open Source increases trust (critical for a security product), enables community audits, and aligns with the Matrix ecosystem.

## Decision
**Open Source after MVP, not before Crypto Spike.**

- Repository prepared from day 0 without secrets
- Reproducible builds planned for later
- ADRs and Threat Model published from the start
- License: Apache 2.0 (compatible with Vodozemac and matrix-sdk-crypto)

## Consequences
- **Positive:** Community trust, external audits, contributor potential
- **Positive:** Forces clean design and disciplined secret management from day 0
- **Negative:** Public scrutiny of early design decisions (mitigated by going public after MVP stabilisation)
