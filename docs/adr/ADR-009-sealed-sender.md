# ADR-009: Sealed Sender

**Status:** ✅ Approved
**Date:** 2026-05-08

## Context
In MVP, the server can see sender_id and recipient_id for every message (metadata visible). Sealed Sender encrypts the sender identity so the server knows only the recipient — reducing metadata leakage but adding cryptographic overhead.

## Decision
**Sealed Sender is deferred past MVP. Metadata remains visible to the server in MVP.**

- MVP message relay: server knows sender, recipient, timestamp, approximate size
- This metadata is already visible in comparable messengers (Signal also knows sender → recipient in basic mode)
- Sealed Sender requires pre-distribution of sender keys or a trusted key directory
- Server-side metadata visible in MVP: `{ sender_user_id, recipient_user_id, timestamp, message_size_approx, delivery_status }`
- Server cannot read message plaintext in any case

## Alternatives Considered

| Alternative | Reason for Rejection |
|---|---|
| **Signal-style Sealed Sender** | Requires signed key pre-distribution or a trusted key directory — both add server complexity; key management is not yet mature enough in MVP |
| **Encrypt sender identity in message envelope** | Would require every recipient to try decrypting every incoming pre-key message with all known sender sessions — computationally expensive and does not scale without a key directory |
| **Metadata-minimising relay (mix-net style)** | Far too complex for MVP; Tor/I2P-style routing is a separate research area outside project scope |
| **Accept metadata visibility** | Chosen for MVP — the server already cannot read message content; metadata visibility is an acceptable trade-off for simplicity at this stage |

## Consequences
- **Positive:** Simpler MVP — no signed key pre-distribution or key directory needed
- **Positive:** Server can perform delivery optimisations with full metadata
- **Negative:** Server operator learns who talks to whom (metadata exposure)
- **Future:** Sealed Sender evaluated post-MVP; requires design around key directory trust model

## References
- Signal Sealed Sender: https://signal.org/blog/sealed-sender/
- ADR-007: Message TTL (related server-side message policy)
- Trust boundary diagram in PROJECT_MAP.md
