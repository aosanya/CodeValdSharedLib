# RelationshipDefinition — Storage, Validation, Inverse, and Soft Delete

> Part of the `SHAREDLIB-011` design. See [index](Owl.md) for all files in
> this series.

---

## 1. Storage and Write Strategy

### 1.1 Unified edge storage

**All** relationships — both `ToMany = true` and `ToMany = false` — are stored
as edge documents in the service's ArangoDB edge collection (e.g.
`agency_relationships`). There is no "store as a property field" path.

Edge document shape:

```json
{
  "_from":    "agency_entities/goal-xyz",
  "_to":      "agency_entities/agency-abc",
  "name":     "belongs_to_agency",
  "agencyID": "agency-abc",
  "deleted":  false,
  "deletedAt": null
}
```

> **Field name**: the edge label is stored under `"name"`, matching
> `Relationship.Name` in the Go struct. There is no `"label"` field.

The `"name"` field is indexed for efficient traversal and filtered lookups.

### 1.2 Write strategy per cardinality

| `ToMany` | `CreateRelationship` behaviour |
|---|---|
| `true` | **Insert** — adds a new independent edge document |
| `false` | **Upsert** — replaces the existing edge with `name == rd.Name && _from == fromID` if one exists; otherwise inserts |

The upsert for `ToMany = false` is atomic at the ArangoDB level:

```aql
UPSERT { _from: @from, name: @name }
INSERT { _from: @from, _to: @to, name: @name, agencyID: @agencyID, deleted: false, deletedAt: null }
UPDATE { _to: @to, deletedAt: null, deleted: false }
IN agency_relationships
```

No read-before-write or distributed lock is needed.

### 1.3 Fetching related entity properties

`CreateRelationship` stores only the edge (IDs + label). To read the
properties of a related entity the caller makes two calls:

1. `ListRelationships(filter: {FromID: entityID, Name: "belongs_to_agency"})` →
   returns the edge(s) and their `ToID`
2. `GetEntity(agencyID, toID)` → returns the full entity with all properties

`GetRelationship`, `ListRelationships`, and `TraverseGraph` work identically
for both `ToMany = true` and `ToMany = false` edges.

---

## 2. Validation — Shared Helpers

Validation logic lives in the `entitygraph` package. Every backend calls these
functions — logic lives once in SharedLib.

### 2.1 `ValidateCreateRelationship`

```go
// ValidateCreateRelationship checks that the proposed edge is permitted by
// the schema. Must be called by every DataManager backend before writing.
//
// Rules enforced:
//  1. label must match a RelationshipDefinition.Name on fromTypeDef.
//  2. toTypeID must equal RelationshipDefinition.ToType.
//
// Cardinality (ToMany=false upsert vs. ToMany=true insert) is handled by
// the backend write strategy — not by this function.
//
// Returns ErrInvalidRelationship if either rule is violated.
func ValidateCreateRelationship(fromTypeDef types.TypeDefinition, label, toTypeID string) error
```

### 2.2 `ValidateSchema`

```go
// ValidateSchema checks the internal consistency of a Schema before it is
// persisted by SetSchema. Called inside SetSchema — invalid schemas are
// rejected and never reach the database.
//
// Rules enforced:
//  1. All TypeDefinition.Name values are unique within the schema.
//  2. For every RelationshipDefinition where Inverse != "":
//     a. ToType must reference a TypeDefinition.Name in the same schema.
//     b. The ToType's TypeDefinition must declare a RelationshipDefinition
//        with Name == rd.Inverse.
//
// Returns a descriptive error on the first violation found.
func ValidateSchema(schema types.Schema) error
```

### 2.3 `CreateEntity` with inline relationships

`CreateEntityRequest` carries an optional `Relationships` field — a typed slice
of `EntityRelationshipRequest`. This allows the entity and all its required
relationships to be created atomically in a single transaction.

```go
// EntityRelationshipRequest carries a single relationship to create
// alongside a new entity in CreateEntityRequest.
type EntityRelationshipRequest struct {
    // Name is the edge label — must match a RelationshipDefinition.Name
    // declared on the entity's TypeDefinition.
    Name string

    // ToID is the target entity ID.
    ToID string
}

// CreateEntityRequest — updated
type CreateEntityRequest struct {
    AgencyID      string
    TypeID        string
    Properties    map[string]any
    Relationships []EntityRelationshipRequest // optional inline edges
}
```

**Transaction order** for `CreateEntity` when `Relationships` is non-empty:

1. Look up the `TypeDefinition` for `TypeID` from the agency's current schema.
2. For each entry in `Relationships`, call `ValidateCreateRelationship` — same
   validation as the standalone `CreateRelationship` path (Q14). Abort before
   any writes if validation fails.
3. Insert the entity document.
4. Write all inline edges (upsert for `ToMany=false`, insert for `ToMany=true`)
   and their auto-inverse counterparts — all in the same transaction.
5. After all edges are written, verify that every `RelationshipDefinition` with
   `Required = true` on this `TypeDefinition` has at least one edge in the
   request. If any are missing, roll back the entire transaction and return
   `ErrRequiredRelationshipViolation` (Q12).

**Empty `Relationships` field** is only valid when the entity's `TypeDefinition`
declares no `Required = true` relationships. If required relationships exist and
none are supplied, the transaction is rolled back at step 5.

---

## 3. Inverse Edge Behaviour

### 3.1 Auto-creation on `CreateRelationship`

When `RelationshipDefinition.Inverse != ""`, the backend writes **two** edge
documents in a **single ArangoDB transaction**:

1. The forward edge: `FromID → Name → ToID`
2. The inverse edge: `ToID → rd.Inverse → FromID`

Both writes succeed or both are rolled back. The caller makes one
`CreateRelationship` call; the inverse edge is transparent.

The inverse `RelationshipDefinition` is looked up via
`FindRelationshipDef(toTypeDef, rd.Inverse)`. Because `ValidateSchema` already
confirmed it exists, this lookup cannot fail at runtime.

### 3.2 Auto-deletion on `DeleteRelationship`

When `DeleteRelationship` is called on a forward edge whose
`RelationshipDefinition.Inverse != ""`, the backend soft-deletes **both** the
forward and inverse edge documents in a **single ArangoDB transaction**.

Backend steps:
1. Read the forward edge document → get `Name`, `FromID`, `ToID`
2. Look up the inverse edge (`name == rd.Inverse`, `_from == ToID`, `_to == FromID`)
3. Soft-delete both in one transaction

If the inverse edge is already soft-deleted, the transaction still succeeds —
the forward edge is soft-deleted regardless.

---

## 4. Soft Delete

**All deletes are soft deletes.** Hard deletion is never performed on entities
or edges.

### 4.1 Soft-delete fields

Both entity documents and edge documents carry:

```json
{ "deleted": false, "deletedAt": null }
```

`DeleteEntity` and `DeleteRelationship` set `deleted = true` and
`deletedAt = <UTC now>`.

### 4.2 `DeleteEntity` cascade

`DeleteEntity` soft-deletes the entity **and** all edges where
`_from == entityID` or `_to == entityID` (including inverse counterparts) in a
**single ArangoDB transaction**.

**Pre-check before the transaction begins (Q15):** Scan all inbound edges
(`_to == entityID`). For each such edge, look up the source entity's
`TypeDefinition` and find the `RelationshipDefinition` matching the edge's
`Name`. If that definition has `Required = true`, deleting entity A would leave
entity B without its required relationship — return
`ErrRequiredRelationshipViolation` and abort (no writes are made).

Order of operations within the transaction (only reached if pre-check passes):
1. Soft-delete all edges where `_from == entityID` (forward edges the entity owns)
2. For each such edge where `rd.Inverse != ""`, soft-delete the corresponding inverse edge
3. Soft-delete all remaining edges where `_to == entityID` (inbound edges not yet covered)
4. Soft-delete the entity document itself

### 4.3 Read exclusion

`GetRelationship`, `ListRelationships`, `TraverseGraph`, `GetEntity`, and
`ListEntities` all filter out documents where `deleted == true`. Soft-deleted
entities and edges are invisible to all read operations.

---

## 5. `TraverseGraph` Edge Name Filter

`TraverseGraphRequest` exposes a `Names []string` field that restricts
traversal to edges whose `Name` is in the list. An empty (or nil) slice means
no filtering — all reachable edges are followed regardless of label.

```go
// TraverseGraphRequest — updated
type TraverseGraphRequest struct {
    AgencyID  string
    StartID   string
    Direction string   // "outbound", "inbound", or "any"
    Depth     int      // 0 treated as 1
    Names     []string // optional — restrict to these edge labels; empty = all
}
```

Backend AQL example (outbound, names filter active):

```aql
FOR v, e, p IN 1..@depth OUTBOUND CONCAT('agency_entities/', @startID)
  GRAPH 'agency_graph'
  FILTER LENGTH(@names) == 0 OR e.name IN @names
  FILTER v.deleted == false
  RETURN DISTINCT v
```

The filter applies in both directions when `Direction = "any"`. When `Names` is
non-empty, only edges whose label is in the list are followed — vertices
reachable exclusively through non-listed edges are not included.

---

*Last updated: 2026-03-19*
