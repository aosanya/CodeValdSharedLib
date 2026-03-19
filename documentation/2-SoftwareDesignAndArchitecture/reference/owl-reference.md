# OWL / RDF Reference Summary

> Compact reference of OWL and RDF constructs used in the
> `RelationshipDefinition` design. See [index](Owl.md) for context.

---

## Construct Mapping

| OWL construct | entitygraph equivalent |
|---|---|
| `owl:DataProperty` | `types.PropertyDefinition` (scalar values) |
| `owl:ObjectProperty` | `types.RelationshipDefinition` (entity references) |
| `rdfs:domain C` | `RelationshipDefinition` declared on `TypeDefinition` C |
| `rdfs:range C` | `RelationshipDefinition.ToType = "C"` |
| `rdfs:subClassOf` | Subtype / inheritance (not yet modelled in entitygraph) |
| `owl:minCardinality 0` | `Required = false` |
| `owl:minCardinality 1` | `Required = true` |
| `owl:maxCardinality 1` | `ToMany = false` — upsert write strategy |
| unbounded cardinality | `ToMany = true` — insert write strategy |
| `owl:inverseOf` | `RelationshipDefinition.Inverse` — auto-created and auto-deleted in a single ArangoDB transaction |

---

## Array Representation

OWL has no native array type. Multiplicity is expressed via cardinality
restrictions on `ObjectProperty`. This maps directly to `ToMany = true` in
the entitygraph — each member of the collection is an independent edge
document in ArangoDB.

---

## Key OWL 2 Profiles (for reference)

| Profile | Logic basis | Typical use |
|---|---|---|
| OWL 2 EL | EL++ | Large biomedical ontologies |
| OWL 2 QL | DL-Lite | Ontology-based data access over relational DBs |
| OWL 2 RL | Rules (Datalog) | Scalable inference in RDF stores |
| OWL 2 DL | SROIQ | Full description-logic reasoning (HermiT, FaCT++) |

The entitygraph design corresponds most closely to **OWL 2 RL** — rule-based
validation at write time, no full DL reasoning.

---

## Further Reading

- W3C OWL 2 Primer: <https://www.w3.org/TR/owl2-primer/>
- W3C OWL 2 Profiles: <https://www.w3.org/TR/owl2-profiles/>
- ArangoDB named graphs: <https://docs.arangodb.com/stable/graphs/>

---

*Last updated: 2026-03-19*
