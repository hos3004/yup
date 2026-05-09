# RELEASE_ACCEPTANCE_CHECKLIST.md

Status: Empty checklist for next internal build acceptance
Project status: Internal Alpha with critical blockers

Do not mark items complete without evidence. This checklist is not a Closed Beta readiness checklist.

## Git Hygiene

- [ ] Release candidate is a named branch or commit.
- [ ] `git status --short --branch` is clean or only contains approved QA artifacts.
- [ ] No untracked application, server, Flutter, Rust, Firebase, or migration files.
- [ ] Commit SHA is recorded in QA evidence.
- [ ] Docs match the audited commit.
- [ ] No unrelated M11 or future milestone changes are mixed into M9/M10 acceptance.

## Build Reproducibility

- [ ] `flutter doctor -v` output captured.
- [ ] `flutter devices` output captured.
- [ ] `flutter pub get` passes from clean checkout.
- [ ] Server dependencies resolve from clean checkout.
- [ ] Rust dependencies resolve from clean checkout.
- [ ] Docker server image builds from clean checkout.
- [ ] Android x86_64 debug APK builds.
- [ ] Android arm64-v8a debug APK builds.

## Flutter

- [ ] `dart analyze lib/` passes with no unresolved release-blocking issues.
- [ ] `flutter analyze` passes.
- [ ] `flutter test` passes.
- [ ] App launches on Android emulator.
- [ ] Fresh install flow works.
- [ ] Registration flow works.
- [ ] Restore session flow works if supported.
- [ ] UI does not crash if push is unavailable.
- [ ] Key-change warning UI is reachable and blocks silent send.
- [ ] Clear local data flow works.
- [ ] Logout behavior matches documentation.

## Rust

- [ ] `cargo test` or documented host-toolchain equivalent passes.
- [ ] `cargo +stable-gnu test` passes if GNU workaround is required.
- [ ] `cargo build --release --target x86_64-linux-android` passes.
- [ ] `cargo build --release --target aarch64-linux-android` passes.
- [ ] Generated `libyup_crypto.so` exists for `x86_64`.
- [ ] Generated `libyup_crypto.so` exists for `arm64-v8a`.
- [ ] Existing checked/copied native libraries are reproducible from current Rust source.
- [ ] Tampered ciphertext decrypt fails closed.
- [ ] Fingerprint is order-independent.
- [ ] Fingerprint changes when either identity key changes.

## Server

- [ ] `go test ./... -v -count=1` passes without `DATABASE_URL_TEST`.
- [ ] `go vet ./...` passes.
- [ ] `go build ./cmd/main.go` passes.
- [ ] Server starts with PostgreSQL `DATABASE_URL`.
- [ ] Server starts in documented in-memory mode if still supported.
- [ ] Health endpoint returns `{"status":"ok"}`.
- [ ] Rate limiting is wired to public and authenticated routes.
- [ ] Request size limits are enforced.
- [ ] Error responses use documented JSON shape.

## PostgreSQL

- [ ] `docker compose up -d postgres` starts a healthy DB.
- [ ] Required `DATABASE_URL_TEST=... go test ./... -v -count=1` passes repeatedly.
- [ ] Test DB isolation prevents cross-package clobbering.
- [ ] Users persist after server restart.
- [ ] Key bundles persist after server restart.
- [ ] OTKs are consumed once.
- [ ] `consumed_at` behavior is verified if schema includes it.
- [ ] Pending messages persist after server restart.
- [ ] ACK updates status correctly.
- [ ] Empty message lists return `[]`, not `null`.
- [ ] No plaintext message content is stored.
- [ ] No private/session keys or pickles are stored.
- [ ] Auth tokens are not stored plaintext.
- [ ] Migrations are idempotent.
- [ ] Down migrations exist where required.

## E2EE

- [ ] A to B online message decrypts locally.
- [ ] A to B offline message decrypts when B opens later.
- [ ] B to A reply decrypts locally.
- [ ] Server restart does not break pending delivery.
- [ ] Fetch-without-ACK crash behavior is tested and documented.
- [ ] Duplicate message prevention is verified.
- [ ] Server cannot decrypt message content.
- [ ] Server logs do not contain plaintext.
- [ ] PostgreSQL does not contain plaintext.
- [ ] Key-change warning blocks silent send.
- [ ] Re-verify or accept-new-key flow is explicit.

## Auth

- [ ] Unauthenticated send fails.
- [ ] Unauthenticated fetch fails.
- [ ] Unauthenticated ACK fails.
- [ ] Unauthenticated device registration fails.
- [ ] Sender spoofing fails.
- [ ] Wrong user cannot fetch another user's queue.
- [ ] Wrong user cannot ACK another user's message.
- [ ] Recipient ACK succeeds.
- [ ] Auth token generation uses secure randomness.
- [ ] Auth token storage risk is resolved or explicitly tracked.

## Push

- [ ] Firebase `google-services.json` package matches Android `applicationId`.
- [ ] App builds with Firebase config.
- [ ] App launches if FCM/server credentials are unavailable.
- [ ] Push token registration happens only after auth.
- [ ] Device token upsert works.
- [ ] Token refresh updates server.
- [ ] Foreground push triggers fetch.
- [ ] Background push behavior is implemented or documented as pending.
- [ ] Resume/open notification behavior is implemented or documented as pending.
- [ ] Push payload contains no plaintext.
- [ ] Push payload contains no ciphertext unless explicitly accepted.
- [ ] Invalid token cleanup is implemented or documented as pending.

## Logging

- [ ] Client LogService redacts bearer tokens.
- [ ] Client LogService redacts auth tokens.
- [ ] Client LogService redacts ciphertext fields.
- [ ] Client LogService redacts account/session pickles.
- [ ] Client LogService redacts session keys.
- [ ] Server logs do not include auth headers.
- [ ] Server logs do not include plaintext messages.
- [ ] Server logs do not include private/session keys.
- [ ] FCM errors do not leak device tokens.
- [ ] Test evidence includes log searches for known secret strings.

## Local Storage

- [ ] Auth token stored only in secure storage.
- [ ] Account pickle stored only in secure storage.
- [ ] Session pickles stored only in secure storage.
- [ ] SQLCipher is used for local message DB.
- [ ] SQLCipher passphrase is generated securely.
- [ ] SQLCipher passphrase is stored in secure storage.
- [ ] Local DB cannot be opened without passphrase.
- [ ] Clear local data deletes DB or creates fresh DB with fresh passphrase.
- [ ] Clear local data removes auth token.
- [ ] Clear local data removes account and session pickles.
- [ ] Logout behavior is documented and verified.

## Docs

- [ ] API contract updated to match stabilized implementation.
- [ ] Manual QA runbook evidence collected.
- [ ] E2E scenarios executed or explicitly deferred.
- [ ] PostgreSQL test plan executed.
- [ ] Push test plan executed or pending items tracked.
- [ ] Security documentation matches implementation.
- [ ] Project map matches actual files and status.
- [ ] No docs claim Closed Beta Ready.
- [ ] Remaining blockers are listed honestly.

## Known Blockers

- [ ] No critical build blockers remain.
- [ ] No critical Rust crypto blockers remain.
- [ ] No critical Android/Firebase blockers remain.
- [ ] No critical PostgreSQL persistence blockers remain.
- [ ] No critical auth/authorization blockers remain.
- [ ] No critical E2EE/key-continuity blockers remain.
- [ ] No critical plaintext logging/storage blockers remain.
- [ ] No critical documentation mismatch remains.
