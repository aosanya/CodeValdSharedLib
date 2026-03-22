# CodeValdSharedLib — Architecture

## 1. Purpose

CodeValdSharedLib is a pure **infrastructure library** — it contains no business
logic. Its sole purpose is to eliminate code duplication across CodeVald
microservices by providing a single, versioned home for code that two or more
services share.

**Design principle**: _If a piece of infrastructure code is used by more than one
service, or is reasonably expected to be needed by a future service, it belongs
in CodeValdSharedLib rather than in the individual service._

---

## 2. Packages

### `registrar` — Generic Cross Heartbeat Registrar

Generic `Registrar` that any CodeVald service uses to announce itself to
CodeValdCross and send periodic heartbeat pings.

```go
// New constructs a Registrar. The caller provides all service-specific
// metadata: serviceName, producedTopics, consumedTopics, and declaredRoutes.
// routes is a []types.RouteInfo slice — use schemaroutes.RoutesFromSchema to
// derive these dynamically from a types.Schema, or build them by hand.
// The registrar converts them to []*crossv1.RouteDeclaration (including
// ConstantBindings) before each Register heartbeat.
func New(
    crossAddr, listenAddr, agencyID string,
    serviceName string,
    produces, consumes []string,
    routes []types.RouteInfo,
    pingInterval, pingTimeout time.Duration,
) (Registrar, error)

func (r *Registrar) Run(ctx context.Context)   // blocking; call in a goroutine
func (r *Registrar) Close()
```

Previously each service had its own `internal/registrar/` with an identical
struct. The only per-service differences (service name, topics, routes) are now
constructor arguments.

---

### `serverutil` — gRPC Server Utilities

Common helpers for spinning up a gRPC server.

```go
// NewGRPCServer creates a *grpc.Server pre-wired with:
//   - gRPC health service (grpc_health_v1) set to SERVING
//   - gRPC server reflection (for grpcurl / dynamic proxy)
func NewGRPCServer() (*grpc.Server, *health.Server)

// RunWithGracefulShutdown starts srv on lis, waits for ctx cancellation,
// then drains in-flight RPCs (up to drainTimeout) before forcing a stop.
func RunWithGracefulShutdown(ctx context.Context, srv *grpc.Server, lis net.Listener, drainTimeout time.Duration)

// EnvOrDefault returns os.Getenv(key), falling back to def when unset or empty.
func EnvOrDefault(key, def string) string

// ParseDurationSeconds reads key from the environment as a positive integer
// number of seconds. Falls back to def on parse failure.
func ParseDurationSeconds(key string, def time.Duration) time.Duration

// ParseDurationString reads key from the environment as a Go duration string
// (e.g. "10s"). Falls back to def on parse failure.
func ParseDurationString(key string, def time.Duration) time.Duration
```

---

### `arangoutil` — ArangoDB Connection Bootstrap

One-call helper to create an authenticated ArangoDB `driver.Database`, used by
all services that persist to ArangoDB.

```go
type Config struct {
    Endpoint string // e.g. "http://localhost:8529"
    Username string // default "root"
    Password string
    Database string // database name
}

// Connect opens an ArangoDB connection, authenticates, and returns the
// named database handle (creating it if it does not exist).
func Connect(ctx context.Context, cfg Config) (driver.Database, error)
```

Each service wraps the returned `driver.Database` in its own
collection-level backend — the connection bootstrap is shared, the schema is not.

---

### `gen/go/codevaldcross/v1` — CodeValdCross Proto-Generated Go Code

Single authoritative copy of the Go code generated from
`proto/codevaldcross/v1/*.proto`.

```
CodeValdSharedLib/
├── proto/codevaldcross/v1/        ← .proto source files
│   ├── orchestrator.proto
│   └── ...
└── gen/go/codevaldcross/v1/       ← generated Go stubs
    ├── orchestrator.pb.go
    ├── orchestrator_grpc.pb.go
    └── ...
```

All services import from here:
```go
crossv1 "github.com/aosanya/CodeValdSharedLib/gen/go/codevaldcross/v1"
```

Previously every service that needed to call CodeValdCross carried its own copy
of the generated code. Any change to the Cross proto was a multi-repo update.

---

### `types` — Shared Domain Types

Go domain types that are meaningful across service boundaries. These are pure
data structures — no logic, no dependencies.

```go
// PathBinding maps one URL path-parameter placeholder to the corresponding
// proto field name in a gRPC request message.
type PathBinding struct {
    URLParam string `json:"url_param"` // e.g. "agencyId"
    Field    string `json:"field"`     // e.g. "agency_id"
}

// ConstantBinding injects a hardcoded field value into every gRPC request for
// a route, regardless of the HTTP request content. Used to pass type_id,
// relationship name, and similar values that are fixed at route-declaration
// time — HTTP callers never need to supply them explicitly.
type ConstantBinding struct {
    Field string `json:"field"` // proto field name to inject, e.g. "type_id"
    Value string `json:"value"` // fixed value, e.g. "Goal"
}

// RouteInfo is the metadata for a single HTTP route declared by a downstream
// service at registration time. CodeValdCross uses GrpcMethod, PathBindings,
// and ConstantBindings in its dynamic reverse proxy.
type RouteInfo struct {
    Method           string            `json:"method"`
    Pattern          string            `json:"pattern"`
    Capability       string            `json:"capability,omitempty"`
    GrpcMethod       string            `json:"grpc_method,omitempty"`
    PathBindings     []PathBinding     `json:"path_bindings,omitempty"`
    ConstantBindings []ConstantBinding `json:"constant_bindings,omitempty"`
}

// ServiceRegistration is the Go domain representation of a downstream service's
// registration payload, after decoding from the proto RegisterRequest.
type ServiceRegistration struct {
    ServiceName string
    AgencyID    string
    Addr        string
    Produces    []string
    Consumes    []string
    Routes      []RouteInfo
    LastPing    time.Time
}
```

CodeValdCross stores and queries these types internally. Any future service that
needs to introspect the registry (e.g. CodeValdAgency) imports from here.

---

### `entitygraph` — Entity-Graph Data Manager & Schema Manager

Generic interfaces and models for any CodeVald service that owns a typed,
graph-structured entity store backed by a versioned schema.

Currently consumed by **CodeValdDT** (digital twins) and **CodeValdComm**
(messages/notifications). Both services alias these interfaces locally and
supply their own ArangoDB-backed implementations via constructor injection.

```go
// DataManager is the business-logic entry point for entity lifecycle and graph
// operations. Consumers alias this as their own service-scoped interface
// (e.g. DTDataManager = entitygraph.DataManager).
// Schema operations are not in scope — see SchemaManager.
// Immutable types (TypeDefinition.Immutable == true) reject UpdateEntity with
// ErrImmutableType. Storage routing is driven by TypeDefinition.StorageCollection.
type DataManager interface {
    // Entity operations
    CreateEntity(ctx context.Context, req CreateEntityRequest) (Entity, error)
    GetEntity(ctx context.Context, agencyID, entityID string) (Entity, error)
    UpdateEntity(ctx context.Context, agencyID, entityID string, req UpdateEntityRequest) (Entity, error)
    DeleteEntity(ctx context.Context, agencyID, entityID string) error
    ListEntities(ctx context.Context, filter EntityFilter) ([]Entity, error)

    // Graph operations
    CreateRelationship(ctx context.Context, req CreateRelationshipRequest) (Relationship, error)
    GetRelationship(ctx context.Context, agencyID, relationshipID string) (Relationship, error)
    DeleteRelationship(ctx context.Context, agencyID, relationshipID string) error
    ListRelationships(ctx context.Context, filter RelationshipFilter) ([]Relationship, error)
    TraverseGraph(ctx context.Context, req TraverseGraphRequest) (TraverseGraphResult, error)
}

// SchemaManager is the schema storage contract injected into a concrete DataManager
// implementation. It owns read and write access to the service's schema
// collection (e.g. dt_schemas, comm_schemas).
type SchemaManager interface {
    SetSchema(ctx context.Context, schema types.Schema) error
    GetSchema(ctx context.Context, agencyID string, version int) (types.Schema, error)
    ListSchemaVersions(ctx context.Context, agencyID string) ([]types.Schema, error)
}
```

All associated models (`Entity`, `Relationship`, `CreateEntityRequest`,
`UpdateEntityRequest`, `EntityFilter`, `CreateRelationshipRequest`,
`RelationshipFilter`, `TraverseGraphRequest`, `TraverseGraphResult`) are defined
in this package and imported by both services.

**Exported error variables** (used by consuming services via `errors.Is`):

```go
var (
    ErrEntityNotFound                   = errors.New("entity not found")
    ErrEntityAlreadyExists              = errors.New("entity already exists")
    ErrRelationshipNotFound             = errors.New("relationship not found")
    ErrImmutableType                    = errors.New("entity type is immutable")
    ErrInvalidRelationship              = errors.New("invalid relationship")
    ErrRelationshipCardinalityViolation = errors.New("relationship cardinality violation")
    ErrRequiredRelationshipViolation    = errors.New("required relationship violation")
    ErrSchemaNotFound                   = errors.New("schema not found")
)
```

---

### `schemaroutes` — Schema-Driven Route Generation

Derives a complete set of HTTP [`types.RouteInfo`] entries from a
[`types.Schema`], eliminating hand-maintained per-type route declarations in
each service. Called once at startup; the result is passed directly to the
SharedLib `registrar`.

```go
// RoutesFromSchema generates all HTTP routes for a service backed by
// entitygraph. Each route carries PathBindings and ConstantBindings so
// CodeValdCross injects type_id and relationship name at dispatch time.
func RoutesFromSchema(schema types.Schema, basePath, agencyIDParam, grpcService string) []types.RouteInfo
```

Routes generated **per TypeDefinition** with a non-empty `PathSegment`:

| HTTP | Pattern | gRPC method | ConstantBindings |
|---|---|---|---|
| `GET` | `{basePath}/{type.PathSegment}` | `ListEntities` | `type_id = td.Name` |
| `POST` | `{basePath}/{type.PathSegment}` | `CreateEntity` | `type_id = td.Name` |
| `GET` | `{basePath}/{type.PathSegment}/{td.EntityIDParam}` | `GetEntity` | `type_id = td.Name` |
| `PUT` | `{basePath}/{type.PathSegment}/{td.EntityIDParam}` | `UpdateEntity` | `type_id = td.Name` (mutable only) |
| `DELETE` | `{basePath}/{type.PathSegment}/{td.EntityIDParam}` | `DeleteEntity` | `type_id = td.Name` |

Routes generated **per RelationshipDefinition** with a non-empty `PathSegment`:

| HTTP | Pattern | gRPC method | ConstantBindings |
|---|---|---|---|
| `GET` | `…/{td.EntityIDParam}/{rel.PathSegment}` | `ListRelationships` | `name = rel.Name` |
| `POST` | `…/{td.EntityIDParam}/{rel.PathSegment}` | `CreateRelationship` | `name = rel.Name` |
| `DELETE` | `…/{td.EntityIDParam}/{rel.PathSegment}/{relId}` | `DeleteRelationship` | _(none)_ |

Currently consumed by **CodeValdAgency** (via `DefaultAgencySchema()`). Any
future service backed by `entitygraph` can call this function with its own
schema and gRPC service path.

---

## 3. Package Layout

```
github.com/aosanya/CodeValdSharedLib/
├── go.mod
├── registrar/
│   └── registrar.go          ← Generic Registrar (Run/Close/ping)
├── serverutil/
│   └── serverutil.go         ← NewGRPCServer, RunWithGracefulShutdown,
│                                EnvOrDefault, ParseDuration*
├── arangoutil/
│   └── arangoutil.go         ← Connect(ctx, Config) driver.Database
├── types/
│   ├── types.go              ← PathBinding, RouteInfo, ServiceRegistration
│   └── schema.go             ← PropertyType, TypeDefinition, Schema, …
├── entitygraph/
│   └── entitygraph.go        ← DataManager, SchemaManager interfaces + all models
├── schemaroutes/
│   └── schemaroutes.go       ← RoutesFromSchema: auto-generates RouteInfo slices from types.Schema
├── proto/
│   └── codevaldcross/
│       └── v1/
│           └── orchestrator.proto
└── gen/
    └── go/
        └── codevaldcross/
            └── v1/
                ├── orchestrator.pb.go
                └── orchestrator_grpc.pb.go
```

---

## 4. Dependency Rules

```
CodeValdCross   ─┐
CodeValdGit     ─┼
CodeValdWork    ─┼
CodeValdAgency  ─┼──► CodeValdSharedLib   (no reverse dependency)
CodeValdDT      ─┼
CodeValdComm    ─┘
```

- **CodeValdSharedLib must never import from any CodeVald service.**
- SharedLib's own imports are restricted to the Go standard library, the
  ArangoDB Go driver, gRPC core packages, and the protobuf runtime.
- No business logic, no domain models specific to a single service.

---

## 5. Versioning & Go Module

Module path: `github.com/aosanya/CodeValdSharedLib`

Each consuming service adds it as a `require` entry in its `go.mod`. Because
the workspace uses a monorepo layout, a `replace` directive pointing to the
local path is used during development:

```
// go.mod of CodeValdGit, CodeValdWork, CodeValdCross
require github.com/aosanya/CodeValdSharedLib v0.0.0

replace github.com/aosanya/CodeValdSharedLib => ../CodeValdSharedLib
```

---

## 6. What Does NOT Belong Here

| Code | Where it lives | Why |
|---|---|---|
| Domain errors (`ErrTaskNotFound`, `ErrRepoNotFound`) | Each service's `errors.go` | Tightly coupled to service-specific types |
| gRPC service handlers (`internal/grpcserver/`) | Each service | Domain-specific request/response mapping |
| Storage collection schemas | Each service's `storage/arangodb/` | Schema is service-specific |
| Proto definitions for each service | Each service's `proto/` | Service-owned contracts |
| `Message`, `Topic`, `FileEntry` | `CodeValdCross/models.go` | Only used inside Cross today; move here if a second consumer appears |
