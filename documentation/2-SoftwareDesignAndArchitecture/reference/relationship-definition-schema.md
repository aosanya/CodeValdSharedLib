# RelationshipDefinition — Problem, OWL Analogy, and Type Extension

> Part of the `SHAREDLIB-011` design. See [index](Owl.md) for all files in
> this series.

---

## 1. The Problem

The current `types.PropertyDefinition` supports only **data properties** —
scalar values like `string`, `integer`, `boolean`, `datetime`. There is no way
inside the schema to express that an `Agency` *has* a collection of `Goal`
entities or a collection of `Workflow` entities.

Relationships between entities already exist at runtime via
`DataManager.CreateRelationship`, but the **schema is silent** about which
entity types may be connected and under what label. This means:

- No validation — any entity can be connected to any other entity
- No introspection — a client cannot discover the graph shape from the schema
- No domain/range enforcement — `Goal` could be connected to a `Workflow`
  without any schema error

The fix is to add **`RelationshipDefinition`** to `TypeDefinition`, mirroring
OWL's `ObjectProperty` concept.

---

## 2. OWL Analogy

| OWL concept | entitygraph equivalent |
|---|---|
| `owl:DataProperty` | `types.PropertyDefinition` (string, integer, …) |
| `owl:ObjectProperty` | `types.RelationshipDefinition` ← **to be added** |
| `rdfs:domain Agency` | `RelationshipDefinition` declared on the `Agency` `TypeDefinition` |
| `rdfs:range Goal` | `RelationshipDefinition.ToType = "Goal"` |
| `owl:minCardinality 0` | `Required = false` |
| `owl:minCardinality 1` | `Required = true` |
| `owl:maxCardinality 1` | `ToMany = false` (upsert semantics) |
| unbounded cardinality | `ToMany = true` (insert each edge independently) |
| `owl:inverseOf` | `Inverse string` field (auto-created and auto-deleted in a transaction) |

In ArangoDB terms, **all** `RelationshipDefinition` edges — regardless of
`ToMany` — are stored as documents in the service's edge collection. The
`DataManager.CreateRelationship` method validates the label against the schema
and applies the correct write strategy (insert vs. upsert) based on `ToMany`.

For the full OWL vocabulary reference see [owl-reference.md](owl-reference.md).

---

## 3. Type Extension

### 3.1 New type — `RelationshipDefinition`

```go
// RelationshipDefinition declares a legal directed edge from the owning
// TypeDefinition to another type within the same Schema.
//
// In OWL terms this is an ObjectProperty with a fixed rdfs:domain (the
// owning TypeDefinition) and rdfs:range (ToType).
type RelationshipDefinition struct {
    // Name is the edge label stored in the ArangoDB edge collection
    // (e.g. "has_goal", "has_work_item"). Must be unique within the
    // owning TypeDefinition.
    Name string

    // Label is the human-readable display name (e.g. "Goals").
    Label string

    // ToType is the TypeDefinition.Name of the target entity class
    // (e.g. "Goal"). Must reference a type declared in the same Schema.
    ToType string

    // ToMany controls cardinality and write strategy.
    //   false → functional (upsert): if an edge with this Name from the same
    //           source already exists it is replaced; otherwise inserted.
    //           Equivalent to owl:maxCardinality 1.
    //   true  → collection (insert): each CreateRelationship call adds a new
    //           independent edge. Equivalent to unbounded cardinality.
    ToMany bool

    // Required indicates that at least one edge of this label must exist on
    // every entity of the owning type (owl:minCardinality 1 when true).
    Required bool

    // Inverse is the optional name of the reciprocal relationship label on the
    // ToType (e.g. "belongs_to_agency"). When set, CreateRelationship writes
    // both the forward and inverse edges in a single transaction.
    // ValidateSchema enforces that the named inverse definition exists on ToType.
    Inverse string
}
```

### 3.2 Updated `TypeDefinition`

Two fields added:

```go
type TypeDefinition struct {
    Name              string
    DisplayName       string
    PathSegment       string                   // URL segment for schema-driven routes, e.g. "goal-templates"
    Properties        []PropertyDefinition
    Relationships     []RelationshipDefinition  // ← NEW
    StorageCollection string
    Immutable         bool
}
```

`PathSegment` is used by the registrar's `GetActive`-driven route generator
(see [schema-versioning.md](schema-versioning.md)). If empty, the type will not
be represented in the generated route set.

---

*Last updated: 2026-03-19*
