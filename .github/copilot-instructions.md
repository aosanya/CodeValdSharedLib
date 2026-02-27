# CodeValdSharedLib — AI Agent Development Instructions

## Project Overview

**CodeValdSharedLib** is a **Go library** that provides shared infrastructure
code for the CodeVald platform. It eliminates code duplication across
[CodeValdCross](../CodeValdCross/README.md),
[CodeValdGit](../CodeValdGit/README.md), and
[CodeValdWork](../CodeValdWork/README.md) by providing a single, versioned
home for code that two or more services share.

**Design principle**: _Any infrastructure code used by more than one service —
or reasonably expected to be needed by a future service — belongs here, not in
the individual service. Individual services own only their domain logic, domain
errors, gRPC handlers, and storage schemas._

**CodeValdSharedLib must never import from any CodeVald service.**

---

## Library Architecture

> **Full architecture details live in the documentation.**
> See `documentation/2-SoftwareDesignAndArchitecture/architecture.md` for:
> - Package layout (`registrar/`, `serverutil/`, `arangoutil/`, `types/`, `gen/`)
> - Dependency rules (SharedLib never imports from services)
> - Versioning and `replace` directive conventions

**Key invariants to keep in mind while coding:**

- **No imports from CodeVald services** — only stdlib, gRPC core, ArangoDB driver, protobuf runtime
- **No domain logic** — infrastructure and plumbing only; no business rules
- All service-specific values (service name, topics, routes) are **constructor arguments**, never hardcoded constants inside SharedLib
- Every exported symbol has a **godoc comment**
- Every exported method takes **`context.Context`** as its first argument

---

## Packages

| Package | Purpose |
|---|---|
| `registrar/` | Generic Cross heartbeat registrar — replaces per-service `internal/registrar/`; caller injects service name, topics, and routes |
| `serverutil/` | gRPC server helpers: `NewGRPCServer`, `RunWithGracefulShutdown`, `EnvOrDefault`, `ParseDurationSeconds`, `ParseDurationString` |
| `arangoutil/` | ArangoDB connection bootstrap: `Connect(ctx, Config) (driver.Database, error)` |
| `types/` | Shared domain types: `PathBinding`, `RouteInfo`, `ServiceRegistration` |
| `gen/go/codevaldcross/v1/` | Authoritative generated Go stubs for the CodeValdCross proto |

---

## Project Structure

```
/workspaces/CodeValdSharedLib/
├── go.mod
├── registrar/
│   └── registrar.go          # Generic Registrar (Run/Close/ping loop)
├── serverutil/
│   └── serverutil.go         # NewGRPCServer, RunWithGracefulShutdown, EnvOrDefault, ParseDuration*
├── arangoutil/
│   └── arangoutil.go         # Connect(ctx, Config) driver.Database
├── types/
│   └── types.go              # PathBinding, RouteInfo, ServiceRegistration
├── proto/
│   └── codevaldcross/
│       └── v1/
│           └── orchestrator.proto
├── gen/
│   └── go/
│       └── codevaldcross/
│           └── v1/
│               ├── orchestrator.pb.go       # Do not hand-edit
│               └── orchestrator_grpc.pb.go  # Do not hand-edit
└── documentation/
    ├── 2-SoftwareDesignAndArchitecture/
    │   └── architecture.md
    └── 3-SofwareDevelopment/
        └── mvp.md
```

---

## Developer Workflows

### Build & Test Commands

```bash
# Build check (library — verifies compilation, no binary produced)
go build ./...

# Run all tests with race detector
go test -v -race ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Static analysis
go vet ./...

# Format code
go fmt ./...

# Lint
golangci-lint run ./...

# Proto regeneration
buf lint && buf generate
```

**There is no `make run`, no binary, no server.** This is a library.

### Task Management Workflow

```bash
# 1. Create feature branch from main
git checkout -b feature/SHAREDLIB-XXX_description

# 2. Implement changes

# 3. Build validation before merge
go build ./...           # Must succeed
go vet ./...             # Must show 0 issues
go test -v -race ./...   # Must pass
golangci-lint run ./...  # Must pass

# 4. Merge when complete
git checkout main
git merge feature/SHAREDLIB-XXX_description --no-ff
git branch -d feature/SHAREDLIB-XXX_description
```

---

## Technology Stack

| Component | Choice | Rationale |
|---|---|---|
| Language | Go 1.21+ | Matches all consuming services |
| gRPC | `google.golang.org/grpc` | Standard gRPC stack for all services |
| Protobuf | `google.golang.org/protobuf` | Code-generated stubs via buf |
| ArangoDB driver | `github.com/arangodb/go-driver` | Used by all services with ArangoDB storage |
| Proto toolchain | buf | Lint and generate protobuf stubs |

---

## Code Quality Rules

### Library-Specific Rules

- **No web framework dependencies** — no Gin, no HTTP handlers, no templating engine
- **No domain logic** — no business rules, no service-specific schemas
- **No imports from CodeVald services** — SharedLib is always a leaf in the dependency graph
- **Interface-first** — package consumers depend on behaviour, not concrete types
- **Exported API is minimal** — expose only what consumers need
- **All public functions must have godoc comments**
- **Context propagation** — every public method takes `context.Context` as first argument
- **Service-specific values are constructor arguments** — service name, topics, routes are never hardcoded

### Naming Conventions

- **Package names**: `registrar`, `serverutil`, `arangoutil`, `types` — lowercase, no abbreviations
- **Interfaces**: noun-only, no `I` prefix
- **Errors**: `Err` prefix for sentinel errors; typed structs when context is needed
- **No abbreviations in exported names** — prefer `AgencyID` over `AgID`
- **Singular package names** — `types`, not `type`

### File Organisation

- **Max file size**: 500 lines (hard limit)
- **Max function length**: 50 lines (prefer 20-30)
- **One primary concern per file**
- **Error types** in each package's own `errors.go` (if needed)

### Anti-Patterns to Avoid

- ❌ **Importing from any CodeVald service** — always the wrong direction
- ❌ **Domain logic or domain errors** — keep in the individual service
- ❌ **Hardcoding service-specific constants** (service names, topic strings) — pass via constructor arguments
- ❌ **Panicking in exported functions** — return structured errors
- ❌ **Ignoring context cancellation** — check `ctx.Err()` in loops
- ❌ **Duplicating code that already exists in SharedLib** — import it

---

## Dependency Rules

```
CodeValdCross   ─┬
                 ├──► CodeValdSharedLib   (no reverse dependency)
CodeValdGit     ─┤
CodeValdWork    ─┘
```

SharedLib's allowed external imports:
- Go standard library
- `google.golang.org/grpc` and `google.golang.org/protobuf`
- `github.com/arangodb/go-driver`
- `google.golang.org/grpc/health/grpc_health_v1`
- `google.golang.org/grpc/reflection`

---

## Documentation References

- `documentation/2-SoftwareDesignAndArchitecture/architecture.md` — package design, dependency rules, versioning
- `documentation/3-SofwareDevelopment/mvp.md` — MVP task list and status (SHAREDLIB-001 through SHAREDLIB-009)

---

## When in Doubt

1. **Check architecture doc first**: `documentation/2-SoftwareDesignAndArchitecture/architecture.md` is the source of truth
2. **Dependency direction**: SharedLib → only stdlib, gRPC, ArangoDB driver — never → CodeVald service
3. **Is it really shared?** If only one service uses it today and no second service is planned, leave it in that service
4. **Inject service-specific values**: constructor arguments, not globals or constants
5. **Write tests for every exported function** — use table-driven tests; aim for >80% coverage
