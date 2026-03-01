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
| SHAREDLIB-006 | `arangoutil` package | 2026-02-27 | `arangoutil/arangoutil.go`: `Config`, `Connect(ctx, Config) driver.Database`; adds `github.com/arangodb/go-driver v1.6.0`; 2 tests pass (`-race`) |

## P2: Consuming Services Migration

| Task ID | Title | Merged | Notes |
|---|---|---|---|
| SHAREDLIB-007 | Migrate CodeValdCross | 2026-02-27 | Removed duplicate `PathBinding`, `RouteInfo`, `ServiceRegistration` from `CodeValdCross/models.go`; replaced with type aliases to `CodeValdSharedLib/types`; added `require github.com/aosanya/CodeValdSharedLib v0.0.0` to `go.mod`; build and vet pass (`-race`) |
| SHAREDLIB-008 | Migrate CodeValdGit | 2026-03-01 | `cmd/main.go` and `cmd/server/main.go` use `sharedregistrar`, `serverutil`, SharedLib's `crossv1` gen; `storage/arangodb` uses `arangoutil.Connect`; removed dead `cmd/cross.go`, local `gen/go/codevaldcross/`, `proto/codevaldcross/`, and empty `internal/registrar/`; all tests pass (`-race`) |
