







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
| [relationship-definition-behaviour.md](relationship-definition-behaviour.md) | Storage and write strategy, validation helpers (`ValidateCreateRelationship`, `ValidateSchema`), inverse edge auto-create/delete, soft delete cascade |
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

---

*Last updated: 2026-03-19*
