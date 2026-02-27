# MVP — CodeValdSharedLib — Completed Tasks

Tasks that have been merged to `master` are moved here from `mvp.md`.

---

## P0: Foundation

| Task ID | Title | Merged | Notes |
|---|---|---|---|
| SHAREDLIB-001 | Go module init | 2026-02-27 | `go mod init github.com/aosanya/CodeValdSharedLib`; `replace` directives added to CodeValdCross, CodeValdGit, CodeValdWork |
| SHAREDLIB-002 | Shared domain types | 2026-02-27 | `types/types.go`: `PathBinding`, `RouteInfo`, `ServiceRegistration`; 5 JSON round-trip tests pass (`-race`) |
