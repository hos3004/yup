# BUG_REPORT_TEMPLATE.md

Use this template for all QA and security-adjacent bugs. Be precise. Attach evidence. Do not file a bug from memory.

## Title

`[Area] Short, specific failure statement`

Example:

`[Android Build] google-services.json package mismatch blocks debug APK`

## Severity

Choose one:

- Critical: Blocks app launch, breaks E2EE/security boundary, causes data loss, allows unauthorized access, stores/leaks secrets, or blocks release acceptance.
- High: Breaks primary messaging, persistence, auth, key verification, local storage safety, or push registration.
- Medium: Breaks secondary workflow, causes flaky tests, creates misleading docs, or requires risky manual workaround.
- Low: Minor functional defect with workaround and low security impact.
- Cosmetic: Text, layout, or presentation issue with no functional/security impact.

Selected severity:

```text

```

## Area

Choose one or more:

- Flutter UI
- Flutter networking
- Flutter local storage
- Flutter push
- Rust crypto
- Go server
- PostgreSQL
- Docker/Makefile
- API contract
- E2E messaging
- Logging/privacy
- Documentation
- Build/CI

Selected area:

```text

```

## Build / Commit

```text
Branch:
Commit SHA:
Dirty working tree? yes/no
Relevant untracked files:
Build artifact:
```

Required commands:

```bash
git status --short --branch
git log --oneline -5
```

## Environment

```text
OS:
Flutter version:
Dart version:
Go version:
Rust toolchain:
Android emulator/device:
Android API level:
Docker version:
PostgreSQL image/version:
Firebase config present? yes/no
GOOGLE_APPLICATION_CREDENTIALS set? yes/no
```

## Preconditions

```text
1.
2.
3.
```

## Steps To Reproduce

Use exact commands, usernames, message text, and request bodies.

```text
1.
2.
3.
4.
```

## Expected Result

```text

```

## Actual Result

```text

```

## Logs

Attach or paste minimal relevant logs. Redact secrets.

```text
Command:
Output:
```

Server logs:

```text

```

Flutter logs:

```text

```

Android logs:

```text

```

PostgreSQL evidence:

```sql
-- query
```

```text
-- output
```

## Screenshots / Recordings

```text
Screenshot path:
Recording path:
```

## API Evidence

Request:

```bash

```

Response:

```http

```

## Security Impact

Answer explicitly:

```text
Does this expose plaintext? yes/no/unknown
Does this expose auth tokens? yes/no/unknown
Does this expose private keys, session keys, or pickles? yes/no/unknown
Does this allow unauthorized access? yes/no/unknown
Does this allow sender spoofing? yes/no/unknown
Does this cause message loss or duplicate delivery? yes/no/unknown
Does this weaken key-change verification? yes/no/unknown
```

Impact explanation:

```text

```

## Regression?

```text
Is this a regression? yes/no/unknown
Last known good commit/build:
Evidence:
```

## Suggested Owner

Choose one:

- Mobile owner
- Server owner
- Crypto owner
- Database owner
- Push/Firebase owner
- QA owner
- Documentation owner
- Unknown

Selected owner:

```text

```

## Blocking Status

Choose one:

- Blocks internal alpha build
- Blocks M9/M10 acceptance
- Blocks Closed Beta
- Blocks Public Beta
- Does not block release
- Needs triage

Selected blocking status:

```text

```

## Workaround

```text
Available workaround:
Risk of workaround:
```

## Attachments Checklist

- [ ] `git status` output
- [ ] command output
- [ ] server logs
- [ ] Flutter logs
- [ ] screenshots or recording
- [ ] SQL query output if persistence-related
- [ ] curl/API transcript if API-related
- [ ] security impact answered
