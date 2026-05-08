# ADR-002: Backend Architecture

**Status:** ✅ Approved (Go version pending local verification)
**Date:** 2026-05-08

## Context
The MVP needs a server to handle auth, key storage, message relay, and delivery. Options evaluated: Go monolith, Elixir/Phoenix.

## Decision
**Go monolith modular.** Phoenix remains a fallback only if team expertise or WebSocket scale demands it.

- Simple deployment (single binary)
- Good enough WebSocket performance for MVP
- Smaller learning curve for a team of 3-5
- Go 1.26 identified as latest stable (1.26.3); **requires local `go version` verification before finalizing**

## Consequences
- **Positive:** Fast iteration, easy CI/CD, one binary to deploy
- **Positive:** stdlib `crypto/hpke` in Go 1.26 may be useful for future features
- **Negative:** Not as natural for WebSocket-heavy apps as Phoenix Channels
- **Negative:** Requires manual WebSocket handling (gorilla/websocket or nhooyr.io/websocket)

## References
- Go 1.26 Release Notes: https://go.dev/doc/go1.26
- Pending: `go version` verification in dev environment
