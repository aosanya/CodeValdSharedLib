# MVP — CodeValdSharedLib

## Goal

Bootstrap `CodeValdSharedLib` as the single home for infrastructure code shared
across CodeVald microservices, then migrate all confirmed candidates out of
CodeValdCross, CodeValdGit, and CodeValdWork.

---

## Design Principle

> **Put as much as possible in the shared lib.**
> Any infrastructure code used by two or more services — or reasonably expected
> to be needed by a future service — belongs here, not in the individual service.
> Individual services own only their domain logic, domain errors, gRPC handlers,
> and storage schemas.

---

## Architecture

See [architecture.md](../2-SoftwareDesignAndArchitecture/architecture.md).

---

## Workflow

### Task Management Process
1. Pick tasks by priority (P0 first) and dependency order.
2. Update Status as work progresses: Not Started → In Progress → Done.
3. **Completion** (MANDATORY):
   - Move completed task row to `mvp_done.md`.
   - Strike through the task ID in dependent tasks: `~~SHAREDLIB-XXX~~ ✅`
   - Merge feature branch to main.

### Branch Naming
```bash
git checkout -b feature/SHAREDLIB-XXX_description
go build ./...
go vet ./...
go test -v -race ./...
git checkout main && git merge feature/SHAREDLIB-XXX_description --no-ff
git branch -d feature/SHAREDLIB-XXX_description
```

---

## Status Legend
- ✅ **Done** — merged to main (see `mvp_done.md`)
- 🚀 **In Progress** — currently being worked on
- 📋 **Not Started** — ready to begin
- ⏸️ **Blocked** — waiting on a dependency

---

## P0: Foundation

| Task ID | Title | Status | Depends On | Notes |
|---|---|---|---|---|
| ~~SHAREDLIB-001~~ ✅ | Go module init | ✅ Done | — | `go mod init github.com/aosanya/CodeValdSharedLib`; add `replace` directives in all consuming services |

---

## P1: Shared Packages

| Task ID | Title | Status | Depends On | Notes |
|---|---|---|---|---|
| ~~SHAREDLIB-002~~ ✅ | Shared domain types | ✅ Done | ~~SHAREDLIB-001~~ ✅ | `types/types.go`: `PathBinding`, `RouteInfo`, `ServiceRegistration` — moved from `CodeValdCross/models.go` |
| ~~SHAREDLIB-003~~ ✅ | CodeValdCross proto-generated code | ✅ Done | ~~SHAREDLIB-001~~ ✅ | Move `.proto` + `gen/go/codevaldcross/v1/` here; single source of truth for all consumers |
| ~~SHAREDLIB-004~~ ✅ | Generic `registrar` package | ✅ Done | ~~SHAREDLIB-001~~ ✅, ~~SHAREDLIB-003~~ ✅ | Move from `CodeValdGit/internal/registrar/` + `CodeValdWork/internal/registrar/`; caller injects `serviceName`, topics, routes |
| ~~SHAREDLIB-005~~ ✅ | `serverutil` package | ✅ Done | ~~SHAREDLIB-001~~ ✅ | `NewGRPCServer()`, `RunWithGracefulShutdown()`, `EnvOrDefault()`, `ParseDurationSeconds()`, `ParseDurationString()` |
| ~~SHAREDLIB-006~~ ✅ | `arangoutil` package | ✅ Done | ~~SHAREDLIB-001~~ ✅ | `Connect(ctx, Config) (driver.Database, error)` — bootstrap only; each service keeps its own collection logic |
| SHAREDLIB-012 | Route write classification (`IsWrite` on `RouteInfo` + `RouteDeclaration`) | 📋 Not Started | ~~SHAREDLIB-002~~ ✅, ~~SHAREDLIB-003~~ ✅ | Service-declared write/read flag per route so Cross's interim Basic-auth gate stops misclassifying POST-search endpoints. Design: [reference/route-write-classification.md](../2-SoftwareDesignAndArchitecture/reference/route-write-classification.md). Spans SharedLib (types + proto + registrar + schemaroutes), Cross (registry + proxy + authMiddleware), Git (annotate every registrar entry). |

---

## P2: Consuming Services Migration

| Task ID | Title | Status | Depends On | Notes |
|---|---|---|---|---|
| ~~SHAREDLIB-007~~ ✅ | Migrate CodeValdCross | ✅ Done | ~~SHAREDLIB-002~~ ✅, ~~SHAREDLIB-003~~ ✅ | Import `types.ServiceRegistration`, `types.RouteInfo`, `types.PathBinding` from SharedLib; remove duplicate definitions from `models.go`; update `go.mod` |
| ~~SHAREDLIB-008~~ ✅ | Migrate CodeValdGit | ✅ Done | ~~SHAREDLIB-003~~ ✅, ~~SHAREDLIB-004~~ ✅, ~~SHAREDLIB-005~~ ✅, ~~SHAREDLIB-006~~ ✅ | Replace `internal/registrar/` with `registrar`; replace `cmd/server/main.go` helpers with `serverutil`; replace ArangoDB bootstrap in `storage/arangodb/` with `arangoutil.Connect`; import Cross gen from SharedLib |
| ~~SHAREDLIB-009~~ ✅ | Migrate CodeValdWork | ✅ Done | ~~SHAREDLIB-003~~ ✅, ~~SHAREDLIB-004~~ ✅, ~~SHAREDLIB-005~~ ✅, ~~SHAREDLIB-006~~ ✅ | Same scope as ~~SHAREDLIB-008~~ ✅ for CodeValdWork |

---

## P3: Entity-Graph Infrastructure

| Task ID | Title | Status | Depends On | Notes |
|---|---|---|---|---|
| ~~SHAREDLIB-010~~ ✅ | `entitygraph` package | ✅ Done |
| ~~SHAREDLIB-011~~ ✅ | RelationshipDefinition — extend `entitygraph` with typed graph edges, schema versioning, and schema-driven HTTP route generation | ✅ Done | ~~SHAREDLIB-001~~ ✅ | `entitygraph/entitygraph.go`: `DataManager` + `SchemaManager` interfaces and all associated models (`Entity`, `Relationship`, `CreateEntityRequest`, `UpdateEntityRequest`, `EntityFilter`, `CreateRelationshipRequest`, `RelationshipFilter`, `TraverseGraphRequest`, `TraverseGraphResult`). ArangoDB-backed concrete implementation. Already consumed by CodeValdDT and CodeValdComm (architecturally defined); now being materialised as Go code for CodeValdAgency refactor. **Completed sub-deliverables**: (a) `types.TypeDefinition.EntityIDParam` — type-specific URL placeholder for entity-ID path segment; when empty, per-entity and relationship routes are skipped; (b) `schemaroutes` package — `RoutesFromSchema(schema, basePath, agencyIDParam, grpcService)` generates the full `[]types.RouteInfo` from a Schema, replacing hand-maintained per-type route lists; (c) `registrar.New` now accepts `[]types.RouteInfo` instead of `[]*crossv1.RouteDeclaration` — proto conversion is internal. |

---

## Success Criteria

- `go build ./...` passes in CodeValdSharedLib, CodeValdCross, CodeValdGit, and CodeValdWork
- `go test -race ./...` all pass in all four modules
- No service contains a local copy of the `registrar` struct, `envOrDefault`, or ArangoDB bootstrap boilerplate
- No service carries its own copy of `gen/go/codevaldcross/v1/`
- `PathBinding`, `RouteInfo`, `ServiceRegistration` are defined exactly once in `CodeValdSharedLib/types/`
- `CodeValdSharedLib` does not import from any CodeVald service
- `entitygraph.DataManager` and `entitygraph.SchemaManager` are defined in SharedLib and importable by CodeValdDT, CodeValdComm, and CodeValdAgency
