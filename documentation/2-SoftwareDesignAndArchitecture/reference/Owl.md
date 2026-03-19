







# RelationshipDefinition Design — Index

> **Task**: `SHAREDLIB-011` — extend `entitygraph` so that entity types can
> formally declare a range of related types (ObjectProperty / graph edges).

This design has been split into focused files per the 300-line documentation
limit.

---

## Files

| File | Contents |
|---|---|
| [relationship-definition-schema.md](relationship-definition-schema.md) | Problem statement, OWL analogy, `RelationshipDefinition` struct, `TypeDefinition` extension |
| [relationship-definition-agency-schema.md](relationship-definition-agency-schema.md) | Full `DefaultAgencySchema` example with all relationships declared; entity graph diagram |
| [relationship-definition-behaviour.md](relationship-definition-behaviour.md) | Storage and write strategy, validation helpers, inverse edge auto-create/delete, soft delete cascade, `TraverseGraph` name filter |
| [schema-versioning.md](schema-versioning.md) | Draft/published collections, `SchemaManager` interface, schema activation, schema-driven HTTP routes, `PathSegment` |
| [owl-reference.md](owl-reference.md) | Compact OWL/RDF construct mapping, array representation, OWL 2 profile summary, further reading |

---

## Key Decisions

| # | Decision |
|---|---|
| Q1 | ArangoDB edge label stored as `"name"` — matches `Relationship.Name` in Go |
| Q2 | Validation logic lives in shared `entitygraph` helpers (`ValidateCreateRelationship`, `ValidateSchema`); backends call them |
| Q3 | `ToMany = false` cardinality enforced by upsert write strategy — no count query needed |
| Q4 | Both `ToMany = true` and `ToMany = false` use edge documents — no property-field path |
| Q5 | `GetRelationship`/`ListRelationships` handle both; functional relationships surface via `ListRelationships` then `GetEntity` |
| Q6 | `Inverse` triggers auto-creation of the back-link edge on `CreateRelationship` |
| Q7 | Forward + inverse edges written in a single ArangoDB transaction |
| Q8 | `ValidateSchema` (called inside `SetSchema`) rejects schemas where `Inverse` references a non-existent back-link definition |
| Q9 | `DeleteRelationship` auto-soft-deletes the inverse edge in the same transaction |
| Q10 | `DeleteEntity` cascades to all edges (`_from` and `_to`) in a single transaction — all soft deletes |
| Q11 | `TraverseGraphRequest.Names []string` filters traversal to edges with matching labels; empty = no filter |
| Q12 | `Required = true` relationships are enforced at `CreateEntity` time — missing required relationships cause rollback |
| Q13 | `CreateEntityRequest.Relationships` is a typed slice of `EntityRelationshipRequest{Name, ToID}` — handles `ToMany=true` (multiple entries for same label) |
| Q14 | `ValidateCreateRelationship` runs on every path — both inline on `CreateEntityRequest.Relationships` and standalone `CreateRelationship` |
| Q15 | `DeleteEntity` is blocked (returns `ErrRequiredRelationshipViolation`) if any incoming edge's source entity declares `Required=true` for that label |
| Q16 | Edges where either endpoint is soft-deleted are excluded from `ListRelationships` and `TraverseGraph` results — edge's own `deleted` field is the sole filter |
| Q17 | `ValidateSchema` is structural only (unique names, valid `Inverse` refs); data-compatibility on schema change is caller responsibility in v1 |
| Q18 | Active schema version tracked via `Active bool` on `types.Schema`; `Activate` flips old→false, new→true in one transaction |
| Q19 | Schema-driven route generation — activating a schema version derives the HTTP route set from `TypeDefinition` + `RelationshipDefinition` and re-registers with CodeValdCross |
| Q20 | Version prefix before agencyID in path: `/v{ver}/{agencyID}/{typeName}/...`; old version routes remain live until explicitly deactivated |
| Q21 | Both entity CRUD routes and per-relationship sub-resource routes are generated from the schema |
| Q22 | `Activate` flips the `Active` pointer only; no automatic data migration — caller backfills before activating |
| Q23 | `SetSchema` stores inactive draft; `Activate` is the explicit promotion step |
| Q24 | `Activate` owns DB flip only; registrar calls `GetActive` on each heartbeat cycle to derive and re-register routes — no explicit trigger needed |
| Q25 | `TypeDefinition.PathSegment string` — schema author sets the URL segment explicitly (e.g. `"goal-templates"`); `GetActive` uses it as-is for route generation |

---

*Last updated: 2026-03-19*
