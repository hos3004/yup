# ADR-006: Groups & Media

**Status:** ✅ Approved
**Date:** 2026-05-08

## Context
MVP scope must be tightly constrained. Group messaging and media/file sharing require significantly more infrastructure: Megolm key distribution for groups, content repository for media, CDN or equivalent for blob storage.

## Decision
**Groups and media are deferred past MVP.**

- MVP is 1:1 text-only messaging
- Group chat requires Megolm (supported by Vodozemac 0.10 but not yet integrated)
- Media requires encrypted upload/download and a content repository
- Both features have known design paths but are intentionally omitted until post-MVP

## Alternatives Considered

| Alternative | Reason for Rejection |
|---|---|
| **Include basic group chat in MVP** | Requires Megolm session fan-out, key distribution, and group membership management — adds weeks of work and a significantly larger attack surface |
| **Include media/file sharing in MVP** | Requires a content repository (S3/MinIO), encrypted upload/download pipeline, and thumbnail generation — outside MVP scope for a text-first messenger |
| **Use Matrix protocol server** | Would solve groups and media but introduces server-side complexity (federation, room state) that is unnecessary for the MVP's 1:1 model |

## Consequences
- **Positive:** Faster MVP delivery, reduced attack surface
- **Positive:** Can design media encryption with knowledge from 1:1 messaging experience
- **Negative:** No group chat or file sharing in initial release
- **Future:** Groups implemented after M5; media after groups; both well-understood from Matrix spec

## References
- Vodozemac Megolm support: https://docs.rs/vodozemac/latest/vodozemac/megolm/
- Matrix spec: https://spec.matrix.org/latest/groups/
