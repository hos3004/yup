# ADR-007: Message TTL

**Status:** ✅ Approved (pending legal review)
**Date:** 2026-05-08

## Context
Undelivered messages accumulate on the server. Without a TTL, storage grows unbounded and old undelivered messages become security debt (ciphertext that may never be fetched but remains accessible).

## Decision
**30-day TTL for undelivered messages.**

- Messages not delivered within 30 days are purged from the server
- Delivered messages may be retained per server policy (subject to future ADR)
- TTL timer starts from message creation timestamp
- Server sends no notification for purged messages — sender is unaware of deletion
- Legal review required: KVKK may impose stricter retention limits
- ⚠️ NOTE: This ADR is approved in principle, but the TTL purge logic is **not yet implemented** in the server (which currently uses in-memory storage). Implementation is blocked on server persistence (see server persistence gap).

## Alternatives Considered

| Alternative | Reason for Rejection |
|---|---|
| **No TTL (retain forever)** | Unbounded storage growth; ciphertext exposure window never closes; KVKK/GDPR storage limitation principle violated |
| **7-day TTL** | Too aggressive — legitimate use cases include users who check messages weekly; 30 days balances privacy and usability |
| **90-day TTL** | Too long for MVP — increases storage cost and ciphertext exposure window without clear benefit |
| **Per-conversation configurable TTL** | Adds UI and server complexity not warranted for MVP; defer to post-MVP |

## Consequences
- **Positive:** Limits server storage cost and ciphertext exposure window
- **Positive:** Natural cleanup mechanism for abandoned conversations
- **Negative:** Legitimate delayed delivery beyond 30 days results in message loss
- **Negative:** Disclosure risk if server operator is compelled to reveal existence of old messages
- **Future:** Configurable TTL or per-conversation expiration may be added post-MVP

## References
- KVKK Article 7 (data retention limits)
- GDPR Article 5(1)(e) (storage limitation principle)
