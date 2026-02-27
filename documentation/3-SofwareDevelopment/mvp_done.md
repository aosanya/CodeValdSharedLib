# MVP — CodeValdSharedLib — Completed Tasks

Tasks that have been merged to `master` are moved here from `mvp.md`.

---

## P0: Foundation

| Task ID | Title | Merged | Notes |
|---|---|---|---|
| SHAREDLIB-001 | Go module init | 2026-02-27 | `go mod init github.com/aosanya/CodeValdSharedLib`; `replace` directives added to CodeValdCross, CodeValdGit, CodeValdWork |
| SHAREDLIB-002 | Shared domain types | 2026-02-27 | `types/types.go`: `PathBinding`, `RouteInfo`, `ServiceRegistration`; 5 JSON round-trip tests pass (`-race`) |

## P1: Shared Packages

| Task ID | Title | Merged | Notes |
|---|---|---|---|
| SHAREDLIB-003 | CodeValdCross proto-generated code | 2026-02-27 | `proto/codevaldcross/v1/registration.proto` + `gen/go/codevaldcross/v1/`; `buf.yaml`/`buf.gen.yaml` added; `go_package` updated to SharedLib import path |
| SHAREDLIB-004 | Generic `registrar` package | 2026-02-27 | `registrar/registrar.go`: exported `Registrar` interface, unexported concrete struct; all service-specific values are constructor args; 4 tests pass (`-race`) |
| SHAREDLIB-005 | `serverutil` package | 2026-02-27 | `serverutil/serverutil.go`: `NewGRPCServer`, `RunWithGracefulShutdown`, `EnvOrDefault`, `ParseDurationSeconds`, `ParseDurationString`; 11 tests pass (`-race`) |
