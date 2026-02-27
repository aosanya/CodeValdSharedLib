---
applyTo: '**'
---

# CodeValdSharedLib — Code Structure Rules

## Library Design Principles

CodeValdSharedLib is a **Go library** — not an application. These rules reflect that:

- **No HTTP handlers, no web framework, no templating engine**
- **No `main` package** — the root is the Go module; each subdirectory is an exported package
- **No imports from CodeVald services** — SharedLib is always a leaf node in the dependency graph
- **Callers inject service-specific values** — service name, topics, routes are constructor arguments, never hardcoded
- **Exported API surface is minimal** — expose only what consumers need

---

## Interface-First Design

**Always define interfaces before concrete types.**

```go
// ✅ CORRECT — exported interface; concrete impl is unexported
type Registrar interface {
    Run(ctx context.Context)
    Close()
}

// ❌ WRONG — leaking a concrete struct to callers
type RegistrarImpl struct {
    conn *grpc.ClientConn
}
```

**File layout — one primary concern per file:**

```
registrar/registrar.go     → Registrar struct, Run, Close, ping loop
serverutil/serverutil.go   → NewGRPCServer, RunWithGracefulShutdown, EnvOrDefault, ParseDuration*
arangoutil/arangoutil.go   → Connect(ctx, Config) driver.Database
types/types.go             → PathBinding, RouteInfo, ServiceRegistration
```

---

## Dependency Rules (CRITICAL)

**CodeValdSharedLib must never import from any CodeVald service.**

```go
// ✅ CORRECT — only stdlib, gRPC, ArangoDB driver, protobuf runtime
import (
    "context"
    "time"
    "google.golang.org/grpc"
    "github.com/arangodb/go-driver"
)

// ❌ WRONG — importing from a CodeVald service
import (
    "github.com/aosanya/CodeValdGit"
    "github.com/aosanya/CodeValdWork"
)
```

SharedLib's allowed external imports:
- Go standard library
- `google.golang.org/grpc` and `google.golang.org/protobuf`
- `github.com/arangodb/go-driver`
- `google.golang.org/grpc/health/grpc_health_v1`
- `google.golang.org/grpc/reflection`

---

## Registrar Package Rules

The `registrar` package is the single generic implementation of the Cross heartbeat registrar.
All service-specific values are constructor arguments — never hardcoded inside the package.

```go
// ✅ CORRECT — all service-specific values are constructor arguments
r, err := registrar.New(
    crossAddr, listenAddr, agencyID,
    "codevaldgit",
    []string{"git.repo.created"},    // produces
    []string{"cross.task.requested"}, // consumes
    declaredRoutes,
    30*time.Second, 10*time.Second,
)

// ❌ WRONG — hardcoding a service name or topic inside registrar package
const serviceName = "codevaldgit"
```

---

## Serverutil Package Rules

```go
// ✅ CORRECT — call site provides all gRPC service registrations
srv, health := serverutil.NewGRPCServer()
mypb.RegisterMyServiceServer(srv, myServiceImpl)
serverutil.RunWithGracefulShutdown(ctx, srv, lis, 5*time.Second)

// ❌ WRONG — registering a service-specific handler inside serverutil
func NewGRPCServer(impl mypb.MyServiceServer) *grpc.Server {
    // registering inside shared lib
}
```

---

## Error Handling Rules

- **Never use `log.Fatal`** in library code — return errors to caller
- **Never panic** in exported functions
- **Wrap errors with context**: `fmt.Errorf("Connect %s: %w", cfg.Database, err)`
- Sentinel errors use `var Err... = errors.New(...)` — only for errors callers need to type-switch on

---

## Context Rules

**Every exported method must accept `context.Context` as the first argument.**

```go
// ✅ CORRECT
func Connect(ctx context.Context, cfg Config) (driver.Database, error)
func (r *Registrar) Run(ctx context.Context)

// ❌ WRONG
func Connect(cfg Config) (driver.Database, error)
```

Respect context cancellation in loops and long-running operations (e.g., the ping retry loop).

---

## Godoc Rules

**Every exported type, function, interface, and method must have a godoc comment.**

```go
// Connect opens an ArangoDB connection, authenticates with the given Config,
// and returns a handle to the named database (creating it if it does not exist).
func Connect(ctx context.Context, cfg Config) (driver.Database, error) {
```

- **Package comment** on the primary file of every package
- **Examples** in `_test.go` files for non-obvious API usage patterns

---

## File Size and Complexity Limits

- **Max file size**: 500 lines (hard limit)
- **Max function length**: 50 lines (prefer 20-30)
- **One primary concern per file**
- **Split a package** into multiple files if a single concern grows beyond 300 lines

---

## Naming Conventions

```go
// ✅ CORRECT — lowercase package names, noun-only interfaces, Err prefix for errors
package registrar
package serverutil
package arangoutil
package types

type Registrar interface{}
var ErrConnect = errors.New("connection failed")

// ❌ WRONG
package Registrar         // uppercase
type IRegistrar interface{} // I prefix
var connectError = ...    // unexported sentinel exposed via behaviour
```

---

## Task Management and Workflow

### Branch Management (MANDATORY)

```bash
# Create feature branch from main
git checkout -b feature/SHAREDLIB-XXX_description

# Implement and validate
go build ./...           # must succeed
go test -v -race ./...   # must pass
go vet ./...             # must show 0 issues
golangci-lint run ./...  # must pass

# Merge when complete
git checkout main
git merge feature/SHAREDLIB-XXX_description --no-ff
git branch -d feature/SHAREDLIB-XXX_description
```

### Pre-Development Checklist

Before adding new code:
1. ✅ Is this type or function already defined elsewhere in SharedLib?
2. ✅ Am I importing only stdlib, gRPC, ArangoDB driver, or protobuf runtime?
3. ✅ Does this function accept `context.Context` as its first argument?
4. ✅ Will the file exceed 500 lines after this change?
5. ✅ Am I injecting service-specific values via constructor arguments?
6. ✅ Does every new exported symbol have a godoc comment?
7. ✅ Is this code genuinely shared (used by ≥2 services), or does it only serve one?

### Code Review Requirements

Every PR must verify:
- [ ] No imports from CodeVald services (`CodeValdGit`, `CodeValdWork`, `CodeValdCross`, etc.)
- [ ] All exported symbols have godoc comments
- [ ] Context propagated through all public calls
- [ ] No files exceeding 500 lines
- [ ] Tests added for all new exported functions
- [ ] `go vet ./...` shows 0 issues
- [ ] `go test -race ./...` passes
- [ ] No service-specific logic or domain models
