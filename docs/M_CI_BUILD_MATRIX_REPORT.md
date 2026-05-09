# M-CI: GitHub Actions CI Build Matrix Report

Date: 2026-05-10

Branch: `m-ci-build-matrix` → master (pending PR)

Status: Internal Alpha gated by automated CI. NOT a Closed Beta readiness claim.

## Summary

GitHub Actions CI matrix is live. Six jobs cover every build/test layer of the project (Go server, Flutter client, Rust crypto, PostgreSQL integration). The matrix runs automatically on every PR to `master` and every push to `master`.

The CI pipeline already paid for itself: it caught a real production bug in the rate limiter that was hidden by Windows' 100 ns timer granularity.

## Final Successful Run

| Field | Value |
|---|---|
| Run ID | 25613732904 |
| Run URL | https://github.com/hos3004/yup/actions/runs/25613732904 |
| Commit | `0de0527` (RateLimiter fix: nanoseconds → seconds) |
| Branch | `m-ci-build-matrix` |
| Conclusion | success |
| Jobs result | **6 / 6 PASS** |

### Per-job result

| # | Job | Conclusion | Duration |
|---|---|---|---|
| 1 | Go · Build & Vet | success | ~20 s |
| 2 | Go · Unit Tests | success | ~12 s |
| 3 | Go · Integration Tests (PostgreSQL) | success | ~50 s |
| 4 | Flutter · Analyze & Test | success | ~80 s |
| 5 | Flutter · Build APK (arm64) | success | ~6 min |
| 6 | Rust · Test & Build | success | ~25 s |

## Action Versions Pinned

| Action | Version | Notes |
|---|---|---|
| `actions/checkout` | `v4` | |
| `actions/setup-go` | `v6` | uses `go-version-file: server/go.mod` |
| `actions/setup-java` | `v4` | temurin 21 (required by Flutter APK build) |
| `actions/cache` | `v4` | cargo registry + target dir |
| `subosito/flutter-action` | `v2` | stable channel, cache enabled |
| `dtolnay/rust-toolchain` | `@stable` | |

Runner: `ubuntu-24.04`. PostgreSQL service container: `postgres:17-alpine` (matches `server/docker-compose.yml`).

`GOTOOLCHAIN: local` is set on every Go job to prevent Go 1.21+ auto-toolchain-download from breaking under cache misses.

## Bug CI Caught (M-CI Already Earned Its Keep)

`server/internal/handler/handler.go` and `server/internal/handler/handler_test.go`:

```go
// Before — wrong
middleware.NewRateLimiter(30, 60)        // production
middleware.NewRateLimiter(1, 60)         // test
```

The second argument is `time.Duration`. Passing a bare `60` makes the rate-limit window **60 nanoseconds**, not 60 seconds.

- On **Windows** the bug was invisible: `time.Now()` has 100 ns granularity, so two consecutive calls fall inside the same tick (`elapsed = 0 ≤ 60 ns`), the visitor counter increments, and the test passes.
- On **Linux** the bug fires every time: `time.Now()` has true nanosecond resolution, so two consecutive calls measure > 60 ns apart, the visitor record is reset on every request, and the limiter degrades to a no-op. `TestRateLimit_Returns429` therefore fails.

In production this meant the 30-requests-per-window limiter was effectively unbounded.

Fix (commit `0de0527`):

```go
// After — correct
middleware.NewRateLimiter(30, 60*time.Second)   // production
middleware.NewRateLimiter(1, time.Minute)       // test
```

This is exactly the kind of platform-dependent failure that local Windows-only verification cannot find. M-CI made it visible on the first Linux run.

## Workflow Triggers (Final)

```yaml
on:
  pull_request:
    branches: [master]
  push:
    branches: [master]
```

Runs on every PR targeting `master` and every push that lands on `master`. The temporary `m-ci-build-matrix` branch trigger used while iterating on this workflow has been removed.

## What CI Does NOT Cover (Out of Scope for M-CI)

- Android NDK cross-compilation for Rust (`aarch64-linux-android`, `x86_64-linux-android`) — requires NDK setup, verified locally before each release.
- Signed release APK — unsigned build only; signing belongs to the release pipeline.
- Two-device E2E smoke test — separate milestone (M-E2E).
- Push delivery against real FCM credentials — separate milestone.

## Merge Gate

The temporary trigger that ran CI on the `m-ci-build-matrix` branch has been removed. Merging `m-ci-build-matrix` into `master` requires:

1. PR opened against `master`
2. CI run on the PR returns 6 / 6 PASS
3. Working tree clean (`git status` empty)
4. No direct push to `master` — merge only via reviewed PR

## Next Milestone

M-AUDIT: independent re-audit of the merged state, then M-E2E (two-device E2EE smoke test with real PostgreSQL persistence).
