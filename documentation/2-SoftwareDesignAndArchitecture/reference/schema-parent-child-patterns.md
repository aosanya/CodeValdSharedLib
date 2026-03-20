````markdown
# Schema Design — Parent-Child Relationship Patterns

> Part of the `SHAREDLIB-011` design. See [index](Owl.md) for all files in
> this series.

---

These rules apply to **all services** that own a pre-delivered schema
(CodeValdAgency, CodeValdComm, CodeValdDT). They were established by reviewing
the agency schema against the `SHAREDLIB-011` entitygraph changes.

---

## Q27 — Empty `Relationships` slice blocks all edges for that type

A `TypeDefinition` with an empty (or nil) `Relationships` slice causes
`ValidateCreateRelationship` to return `ErrInvalidRelationship` for **any**
`CreateRelationship` call whose `FromID` resolves to that type.

**Rule**: every `TypeDefinition` that participates in any graph edge — whether
as source or target — must declare those edges in its `Relationships` slice.
Types that are intentionally terminal (no outbound edges) must still appear as
the `ToType` in another type's `RelationshipDefinition`; they simply carry an
empty `Relationships` slice only if they truly produce no outbound edges.

```go
// ❌ WRONG — Agency can never be connected to Goal
{
    Name: "Agency",
    Relationships: nil, // CreateRelationship returns ErrInvalidRelationship
}

// ✅ CORRECT — Agency declares all outbound edges
{
    Name: "Agency",
    Relationships: []types.RelationshipDefinition{
        {Name: "has_goal", ToType: "Goal", ToMany: true, Inverse: "belongs_to_agency"},
    },
}
```

---

## Q28 — `PathSegment` must be set on every addressable type

`TypeDefinition.PathSegment` is the URL segment used by the registrar's
schema-driven route generator. If empty, **no HTTP routes are generated** for
that type.

**Rule**: set `PathSegment` on every type that should be accessible via the
HTTP API. Leave it empty only when the type is intentionally not addressable
(e.g. a root-context type where the agency itself is the URL scope).

```go
// ✅ CORRECT — goal entities are addressable at /v1/{agencyID}/goals/...
{Name: "Goal", PathSegment: "goals", ...}

// intentional exception — Agency IS the agency context; no top-level routes
{Name: "Agency", PathSegment: "", ...}
```

`RelationshipDefinition.PathSegment` follows the same rule for sub-resource
routes (e.g. `/v1/{agencyID}/workflows/{id}/work-items`).

---

## Q29 — Child types declare their parent relationship as `Required: true, ToMany: false`

Any type that is semantically a child of another type must declare a
`belongs_to_*` inverse relationship with:

- `ToMany: false` — a child has exactly one parent (upsert write strategy)
- `Required: true` — `CreateEntity` must supply this relationship atomically;
  orphan children are rejected with `ErrRequiredRelationshipViolation`

This ensures the graph has no orphan nodes — every child entity is reachable
from its parent from the moment it is created.

```go
// ✅ CORRECT — Goal cannot exist without an Agency
{
    Name: "belongs_to_agency",
    ToType: "Agency",
    ToMany: false,
    Required: true,
}

// ❌ WRONG — Goal can be created as an orphan
{
    Name: "belongs_to_agency",
    ToType: "Agency",
    ToMany: false,
    Required: false, // orphan goals are permitted — wrong
}
```

The parent's corresponding outbound `RelationshipDefinition` declares
`Inverse: "belongs_to_*"` so that `CreateRelationship` writes both edges
atomically in a single transaction (see Q6, Q7).

---

## Q30 — Root / container types carry no `Required` outbound relationships

A root type (Agency, Channel, etc.) can be created without any children. Its
outbound relationships (`has_goal`, `has_workflow`, …) are `ToMany: true` with
`Required: false`.

**Rationale**: the UI creates the root entity first and populates children
progressively. Requiring children at root creation time would force the UI to
collect all data upfront before any save is possible.

```go
// ✅ CORRECT — Agency can be saved at any point; children are optional
{
    Name: "has_goal",
    ToType: "Goal",
    ToMany: true,
    Required: false, // default zero value — no children required at creation
}
```

---

## Q31 — Business completeness is deferred to the publish gate; structural integrity is enforced at `CreateEntity`

Two distinct validation layers exist with different scopes:

| Layer | Enforced by | When | What |
|---|---|---|---|
| **Structural integrity** | `entitygraph` (`ErrRequiredRelationshipViolation`) | `CreateEntity` | Every child has a parent link; no orphan nodes |
| **Business completeness** | Service publish/activation flow | `Publish` / `Activate` | Domain rules (e.g. "an Agency must have at least one Goal before activation") |

**Rule**: do not encode business completeness rules as `Required: true`
relationships on the root type. Use the publish gate instead.

```
UI flow example (Agency setup):
  1. CreateEntity(Agency)          ← saves immediately; no children required
  2. CreateEntity(Goal)            ← must supply belongs_to_agency atomically (Q29)
  3. CreateEntity(Workflow)        ← must supply belongs_to_agency atomically
  4. CreateEntity(WorkItem)        ← must supply belongs_to_workflow atomically
  5. Publish / Activate            ← business completeness validated here
```

This separation lets the UI save progress at any step without hitting
validation errors, while the publish gate remains the definitive correctness
check.

---

*Last updated: 2026-03-20*
````
