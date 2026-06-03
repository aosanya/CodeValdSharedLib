# FEAT-20260603-002 (SharedLib) — Migrate all service `Topic*` constants to use `eventbus.Domain*`

**Status:** 📋 Not Started  
**Severity:** Medium — hardcoded string literals are functional today but will silently diverge if a domain prefix is ever renamed; no compile-time safety
**Owner:** CodeValdSharedLib (root contract) — migration touches CodeValdWork, CodeValdGit, CodeValdAI, CodeValdComm, CodeValdFunctions, CodeValdCross
**Estimated effort:** ~1 day (mechanical sweep across 6 services + compile check)
**Source finding:** Architecture audit 2026-06-03 — Rule 7b: 40+ hardcoded domain-prefix string literals in Topic constants across all services

## Problem

Every service defines `Topic*` constants with hardcoded domain prefix string literals. There is no compile-time guard that prevents a service from drifting to a wrong prefix or typo.

Affected files and approximate counts:

| Service | File | Hardcoded constants |
|---------|------|---------------------|
| CodeValdWork | `events.go` | 22 |
| CodeValdGit | `events.go` | 8 |
| CodeValdAI | `events.go` | 8 |
| CodeValdComm | `events.go` | 7 |
| CodeValdFunctions | `events.go` | 2 |
| CodeValdCross | `internal/pubsub/topics.go` | 12 (spanning `git.*`, `work.*`, `cross.*`) |

## Evidence

```bash
grep -rn 'Topic.*=\s*"work\.\|Topic.*=\s*"git\.\|Topic.*=\s*"ai\.\|Topic.*=\s*"comm\.\|Topic.*=\s*"functions\.' \
  /workspaces/CodeVald-AIProject --include="*.go" | grep -v "_test\.go\|CodeValdSharedLib"
# → 47 hits across the 6 services above
```

```go
// Current (violation)
const TopicTaskCreated = "work.task.created"

// Target (correct)
import "github.com/aosanya/CodeValdSharedLib/eventbus"
const TopicTaskCreated = eventbus.DomainWork + "task.created"
```

## Root cause

[FEAT-20260603-001](FEAT-20260603-001_eventbus-domain-constants.md) (SharedLib `Domain*` constants) was never created, so there was nothing to import. Once that lands, this migration is purely mechanical.

## Fix plan

**Phase 1 — prerequisite**: merge [FEAT-20260603-001](FEAT-20260603-001_eventbus-domain-constants.md).

**Phase 2 — per-service sweep** (can be done in one branch or one per service):

For each service, in its `events.go` (or `internal/pubsub/topics.go` for Cross):

1. Add `"github.com/aosanya/CodeValdSharedLib/eventbus"` to the import block.
2. Replace every `= "<domain>.<rest>"` with `= eventbus.Domain<Name> + "<rest>"`.
3. Run `go build ./...` and `go vet ./...` in the service root.

Example diff for `CodeValdWork/events.go`:
```go
-   TopicTaskCreated = "work.task.created"
+   TopicTaskCreated = eventbus.DomainWork + "task.created"
```

**Phase 3 — CodeValdAI inline strings**: after the events.go migration, address the inline git-domain comparisons in `execute.go` and `event_receiver.go` — see [BUG-20260603-002 (AI)](../../../../CodeValdAI/documentation/3-SofwareDevelopment/bug-details/BUG-20260603-002_inline-hardcoded-git-topic-strings.md).

## Verification

```bash
grep -rn 'Topic.*=\s*"work\.\|Topic.*=\s*"git\.\|Topic.*=\s*"ai\.\|Topic.*=\s*"comm\.\|Topic.*=\s*"functions\.' \
  /workspaces/CodeVald-AIProject --include="*.go" | grep -v "_test\.go\|CodeValdSharedLib"
# → no output
```

`go build ./...` passes in all 6 affected services.

## Dependencies

- Blocked on: [FEAT-20260603-001](FEAT-20260603-001_eventbus-domain-constants.md) — SharedLib `Domain*` constants must exist first
- Enables fix of: [BUG-20260603-002 (AI)](../../../../CodeValdAI/documentation/3-SofwareDevelopment/bug-details/BUG-20260603-002_inline-hardcoded-git-topic-strings.md)
