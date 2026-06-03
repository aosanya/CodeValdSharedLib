# FEAT-20260603-001 (SharedLib) — Create `eventbus/domains.go` with `Domain*` prefix constants

**Status:** 📋 Not Started
**Severity:** High — absence of these constants forces every service to duplicate domain prefix strings; any prefix change requires a multi-repo sweep with no compile-time safety net
**Owner:** CodeValdSharedLib
**Estimated effort:** ~0.5 day (new file + update SharedLib go.mod exports)
**Source finding:** Architecture audit 2026-06-03 — Rule 7a: SharedLib has no `Domain*` constants; Rule 7b: all services use hardcoded string literals

## Problem

`CodeValdSharedLib/eventbus/` does not define domain prefix constants. Every service that declares `Topic*` constants must hardcode the domain prefix as a string literal (e.g. `"work."`, `"git."`, `"ai."`). This duplicates the source of truth and means a prefix rename requires touching every service independently with no compile-time enforcement.

## Evidence

```bash
grep -rn "DomainWork\|DomainGit\|DomainAI\|DomainComm\|DomainFunctions\|DomainAgency\|DomainOrg" \
  /workspaces/CodeVald-AIProject/CodeValdSharedLib --include="*.go"
# → (no output)
```

Current hardcoded pattern in every service:
```go
// CodeValdWork/events.go
const TopicTaskCreated = "work.task.created"  // "work." is duplicated truth
```

## Root cause

`eventbus/domains.go` was never created. When Topic constants were first added to each service, no SharedLib contract existed to import from.

## Fix plan

Create `CodeValdSharedLib/eventbus/domains.go`:

```go
package eventbus

const (
    DomainWork      = "work."
    DomainGit       = "git."
    DomainAI        = "ai."
    DomainComm      = "comm."
    DomainFunctions = "functions."
    DomainAgency    = "agency."
    DomainOrg       = "org."
    DomainCross     = "cross."
    DomainPubSub    = "pubsub."
)
```

No other changes needed in SharedLib itself — this is a pure addition.

## Verification

- `go build ./...` passes in CodeValdSharedLib
- `grep -rn "DomainWork" /workspaces/CodeVald-AIProject/CodeValdSharedLib --include="*.go"` returns a hit in `eventbus/domains.go`
- Each consuming service can import `"github.com/aosanya/CodeValdSharedLib/eventbus"` and reference `eventbus.DomainWork`

## Dependencies

- Must land before [FEAT-20260603-002](FEAT-20260603-002_migrate-topic-constants-to-sharedlib.md) — that feature depends on these constants existing
