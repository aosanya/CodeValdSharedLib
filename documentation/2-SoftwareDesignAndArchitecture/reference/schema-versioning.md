# Schema Versioning and Schema-Driven Routes

> Part of the `SHAREDLIB-011` design. See [index](Owl.md) for all files in
> this series.

---

## 1. Two-Collection Model

Schema state lives in two separate ArangoDB collections per service:

| Collection | Mutable? | Purpose |
|---|---|---|
| `schemas_draft` | ✅ Yes — one document per agency, overwritten by `SetSchema` | Work-in-progress schema being edited |
| `schemas_published` | ❌ No — append-only immutable snapshots | Versioned history; one document becomes `Active` at a time |

**Flow:**

```
SetSchema        →  edit draft in schemas_draft
GetSchema        →  read draft from schemas_draft
Publish          →  snapshot draft → new immutable doc in schemas_published (Active = false)
Activate(ver)    →  flip Active: old version false, target version true (one transaction)
GetActive        →  read single doc where Active == true from schemas_published
GetVersion(ver)  →  read specific version from schemas_published
ListVersions     →  all docs in schemas_published for the agency, ascending version
```

---

## 2. `SchemaManager` Interface

```go
// SchemaManager is the schema storage contract injected into a concrete
// DataManager implementation. Draft and published schemas live in separate
// collections; all five published-collection operations act only on
// schemas_published.
type SchemaManager interface {
    // Draft collection — one mutable document per agency.

    // SetSchema overwrites the agency's current draft schema.
    // The draft is never versioned; only published snapshots are versioned.
    SetSchema(ctx context.Context, schema types.Schema) error

    // GetSchema returns the agency's current draft schema.
    // Returns ErrSchemaNotFound if no draft exists yet.
    GetSchema(ctx context.Context, agencyID string) (types.Schema, error)

    // Published collection — immutable, append-only.

    // Publish snapshots the current draft into schemas_published as a new
    // version with Active = false. The version number is auto-assigned
    // (highest existing version + 1).
    Publish(ctx context.Context, agencyID string) error

    // Activate promotes the given version to active, setting Active = true on
    // the target version and Active = false on any previously active version,
    // in a single transaction.
    // Returns ErrSchemaNotFound if the version does not exist.
    Activate(ctx context.Context, agencyID string, version int) error

    // GetActive returns the single published schema where Active == true.
    // Returns ErrSchemaNotFound if no version has been activated yet.
    GetActive(ctx context.Context, agencyID string) (types.Schema, error)

    // GetVersion returns a specific published version.
    // Returns ErrSchemaNotFound if the version does not exist.
    GetVersion(ctx context.Context, agencyID string, version int) (types.Schema, error)

    // ListVersions returns all published versions for the agency in ascending
    // version order. Includes both active and inactive versions.
    ListVersions(ctx context.Context, agencyID string) ([]types.Schema, error)
}
```

---

## 3. `types.Schema` — Active Field

`types.Schema` gains one field:

```go
type Schema struct {
    ID        string             // agency-scoped schema identifier
    AgencyID  string
    Version   int                // auto-assigned on Publish; 0 for draft
    Active    bool               // true for the single active published version
    Types     []TypeDefinition
    CreatedAt time.Time
}
```

Draft documents always have `Version = 0` and `Active = false`. Only published
documents carry a real version number or `Active = true`.

---

## 4. Write Operations Use the Active Schema

`CreateEntity` and `CreateRelationship` call `GetActive` to fetch the agency's
current schema before running `ValidateCreateRelationship`. If no active schema
exists, both operations return `ErrSchemaNotFound`.

Callers must activate at least one schema version before writing entities or
relationships for an agency.

---

## 5. Schema-Driven HTTP Routes

### 5.1 Route generation

When the registrar heartbeat fires, it calls `GetActive` to derive the full
HTTP route set for the agency. Routes are generated from `TypeDefinition` and
`RelationshipDefinition` entries in the active schema.

A `TypeDefinition` with a non-empty `PathSegment` produces six routes:

| Method | Pattern | Maps to |
|---|---|---|
| `POST` | `/v{ver}/{agencyID}/{pathSegment}` | `CreateEntity` |
| `GET` | `/v{ver}/{agencyID}/{pathSegment}` | `ListEntities` |
| `GET` | `/v{ver}/{agencyID}/{pathSegment}/{id}` | `GetEntity` |
| `PATCH` | `/v{ver}/{agencyID}/{pathSegment}/{id}` | `UpdateEntity` |
| `DELETE` | `/v{ver}/{agencyID}/{pathSegment}/{id}` | `DeleteEntity` |

Each `RelationshipDefinition` on that type adds three sub-resource routes
(pending Q26 — `PathSegment` field on `RelationshipDefinition`):

| Method | Pattern | Maps to |
|---|---|---|
| `POST` | `/v{ver}/{agencyID}/{typeSeg}/{id}/{relSeg}` | `CreateRelationship` |
| `GET` | `/v{ver}/{agencyID}/{typeSeg}/{id}/{relSeg}` | `ListRelationships` |
| `DELETE` | `/v{ver}/{agencyID}/{typeSeg}/{id}/{relSeg}/{relID}` | `DeleteRelationship` |

`{ver}` is the schema version number (e.g. `v2`). `{pathSegment}` comes from
`TypeDefinition.PathSegment`.

### 5.2 Version coexistence

Activating schema v2 registers a new route set under `/v2/...`. The `/v1/...`
routes remain live in CodeValdCross until `Activate` is called to explicitly
deactivate them (or a grace-period policy removes them). Both versions can
serve traffic simultaneously during migration.

### 5.3 Route generation trigger

The registrar heartbeat (every 20 s) calls `GetActive` and derives the route
set on every tick. `Activate` only flips the DB pointer — no explicit
re-registration call is required. The new routes go live within one heartbeat
cycle of activation.

---

## 6. `TypeDefinition.PathSegment`

```go
type TypeDefinition struct {
    Name              string
    DisplayName       string
    PathSegment       string                    // URL segment, e.g. "goal-templates"
    Properties        []PropertyDefinition
    Relationships     []RelationshipDefinition
    StorageCollection string
    Immutable         bool
}
```

Rules:
- Must be lowercase, hyphen-separated (e.g. `"goals"`, `"goal-templates"`)
- Must be unique within the schema
- If empty, the type is **not** represented in the generated route set
- Changing `PathSegment` in a new schema version changes the URL — old routes
  for the old version remain live until that version is deactivated

---

*Last updated: 2026-03-19*
