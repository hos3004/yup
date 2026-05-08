# ADR-010: Phone Discovery

**Status:** ✅ Approved
**Date:** 2026-05-08

## Context
Phone number discovery (find contacts by phone number) is the most common social graph feature in messaging apps, but creates privacy risks (phone enumeration, contact data exposure) and regulatory obligations (KVKK, GDPR).

## Decision
**Phone discovery is optional post-MVP with strict controls. Not in MVP.**

- MVP uses username + QR/invite link for user discovery
- Phone discovery considered only post-MVP and only under these conditions:
  - Rate-limited to prevent enumeration
  - Abuse detection (flag bulk lookups)
  - No presence disclosure (requester does not learn if a number is registered unless both are contacts)
  - User opt-in required (default off)
  - Contact hashes stored, not plain phone numbers
- Private Contact Discovery (Signal-style) evaluated as alternative to naive hash lookup

## Alternatives Considered

| Alternative | Reason for Rejection |
|---|---|
| **Phone-only identity** | Rejected in ADR-001; phone numbers are not suitable as primary identity due to KVKK constraints, SIM-swap risks, and privacy concerns |
| **Naive phone hash lookup** | Vulnerable to enumeration and rainbow-table attacks on low-entropy phone numbers; even hashed phone numbers can be reversed |
| **Signal-style Private Contact Discovery** | Requires trusted hardware (Intel SGK) or a privacy-preserving protocol — significant infrastructure investment for post-MVP evaluation only |
| **Include phone discovery in MVP** | Adds KVKK compliance burden, abuse potential, and UI complexity — not justified for closed beta with invitation-based growth |
| **QR/invite-only discovery** | Chosen for MVP — simplest model with zero phone number data processing |

## Consequences
- **Positive:** No phone number data processing in MVP — simpler compliance
- **Positive:** No phone enumeration risk in initial release
- **Negative:** Higher friction for user discovery (must exchange username/QR out-of-band)
- **Future:** Privacy-preserving contact discovery remains an open research area; may use Intel SGX or similar if required

## References
- KVKK constraints on phone data processing
- Signal Private Contact Discovery: https://signal.org/blog/private-contact-discovery/
- ADR-001: Identity Model (username-based discovery for MVP)
