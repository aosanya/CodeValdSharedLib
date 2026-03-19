# RelationshipDefinition — DefaultAgencySchema Example

> Part of the `SHAREDLIB-011` design. See [index](Owl.md) for all files in
> this series.

---

This file shows the full `DefaultAgencySchema` updated with
`RelationshipDefinition`. For the type definition itself see
[relationship-definition-schema.md](relationship-definition-schema.md).

---

## `DefaultAgencySchema` with Relationships

```go
func DefaultAgencySchema() types.Schema {
    return types.Schema{
        ID:      "agency-schema-v1",
        Version: 1,
        Tag:     "v1",
        Types: []types.TypeDefinition{
            {
                Name:        "Agency",
                DisplayName: "Agency",
                Properties: []types.PropertyDefinition{
                    {Name: "name",    Type: types.PropertyTypeString, Required: true},
                    {Name: "mission", Type: types.PropertyTypeString, Required: false},
                    {Name: "vision",  Type: types.PropertyTypeString, Required: false},
                    {Name: "status",  Type: types.PropertyTypeString, Required: true},
                },
                Relationships: []types.RelationshipDefinition{
                    {Name: "has_goal",           Label: "Goals",            ToType: "Goal",              ToMany: true},
                    {Name: "has_workflow",        Label: "Workflows",        ToType: "Workflow",          ToMany: true},
                    {Name: "has_configured_role", Label: "Configured Roles", ToType: "ConfiguredRole",   ToMany: true},
                    {Name: "has_snapshot",        Label: "Snapshots",        ToType: "AgencySnapshot",   ToMany: true},
                    {Name: "has_publication",     Label: "Publications",     ToType: "AgencyPublication", ToMany: true},
                },
            },
            {
                Name:        "Goal",
                DisplayName: "Goal",
                Properties: []types.PropertyDefinition{
                    {Name: "title",       Type: types.PropertyTypeString,  Required: true},
                    {Name: "description", Type: types.PropertyTypeString,  Required: false},
                    {Name: "ordinality",  Type: types.PropertyTypeInteger, Required: true},
                },
                Relationships: []types.RelationshipDefinition{
                    // ToMany=false: a Goal belongs to exactly one Agency.
                    // CreateRelationship upserts — replacing any prior edge.
                    {Name: "belongs_to_agency", Label: "Agency", ToType: "Agency", ToMany: false},
                },
            },
            {
                Name:        "Workflow",
                DisplayName: "Workflow",
                Properties: []types.PropertyDefinition{
                    {Name: "name", Type: types.PropertyTypeString, Required: true},
                },
                Relationships: []types.RelationshipDefinition{
                    {Name: "has_work_item",     Label: "Work Items", ToType: "WorkItem", ToMany: true},
                    {Name: "belongs_to_agency", Label: "Agency",     ToType: "Agency",   ToMany: false},
                },
            },
            {
                Name:        "WorkItem",
                DisplayName: "Work Item",
                Properties: []types.PropertyDefinition{
                    {Name: "title",       Type: types.PropertyTypeString,  Required: true},
                    {Name: "description", Type: types.PropertyTypeString,  Required: false},
                    {Name: "order",       Type: types.PropertyTypeInteger, Required: true},
                    {Name: "parallel",    Type: types.PropertyTypeBoolean, Required: false},
                },
            },
            {
                Name:        "ConfiguredRole",
                DisplayName: "Configured Role",
                Properties: []types.PropertyDefinition{
                    {Name: "name",       Type: types.PropertyTypeString, Required: true},
                    {Name: "actor_type", Type: types.PropertyTypeString, Required: true},
                },
            },
            {
                Name:              "AgencySnapshot",
                DisplayName:       "Agency Snapshot",
                Immutable:         true,
                StorageCollection: "agency_snapshots",
                Properties: []types.PropertyDefinition{
                    {Name: "snapshot_at", Type: types.PropertyTypeDatetime, Required: true},
                },
            },
            {
                Name:              "AgencyPublication",
                DisplayName:       "Agency Publication",
                Immutable:         true,
                StorageCollection: "agency_publications",
                Properties: []types.PropertyDefinition{
                    {Name: "version",      Type: types.PropertyTypeInteger,  Required: true},
                    {Name: "tag",          Type: types.PropertyTypeString,   Required: true},
                    {Name: "published_at", Type: types.PropertyTypeDatetime, Required: true},
                },
            },
        },
    }
}
```

---

## Relationship Graph (Agency schema)

```
Agency ──has_goal──────────────► Goal
       ──has_workflow──────────► Workflow ──has_work_item──► WorkItem
       ──has_configured_role──► ConfiguredRole
       ──has_snapshot─────────► AgencySnapshot   (Immutable)
       ──has_publication──────► AgencyPublication (Immutable)

Goal       ──belongs_to_agency──► Agency   (ToMany=false, upsert)
Workflow   ──belongs_to_agency──► Agency   (ToMany=false, upsert)
```

`belongs_to_agency` edges are functional — at most one per entity, enforced
by upsert write strategy. All other edges are collections (`ToMany=true`).

---

*Last updated: 2026-03-19*
