# ADR-001: Identity Model

**Status:** ✅ Approved
**Date:** 2026-05-08

## Context
The MVP needs an identity model for user registration and discovery. Options were phone-only, username-only, or hybrid.

## Decision
**Username + QR/Invite link for Closed Beta.** Phone-only rejected as final identity model for MVP.

- The identity model in Closed Beta is **username-based**
- Users share their identity via QR code or invite link
- No phone number required
- No bulk contact upload
- No open endpoint for phone number discovery

## Consequences
- **Positive:** Avoids phone enumeration risk, stronger privacy by default, simpler compliance
- **Negative:** Higher friction for user discovery (no automatic contact matching)
- **Future:** Phone number may be added as optional later, with strict rate limiting, abuse detection, and no direct presence disclosure

## References
- KVKK constraints on phone data processing
- Signal's phone-based model rejected for privacy reasons in this project
