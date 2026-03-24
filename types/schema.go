package types

import "time"

// PropertyType is the data type of a single property in a [TypeDefinition].
//
// Primitive types map directly to Go/JSON primitives.
// Choice types (option, select, multiselect) declare the field shape; the
// allowed option values are stored as runtime data in the service's own
// collection, not baked into the schema definition.
// Complex types carry additional configuration on [PropertyDefinition].
type PropertyType string

const (
	// Primitive types.

	// PropertyTypeString is a UTF-8 text value.
	PropertyTypeString PropertyType = "string"

	// PropertyTypeInteger is a 64-bit signed integer.
	PropertyTypeInteger PropertyType = "integer"

	// PropertyTypeFloat is a 64-bit floating-point number.
	PropertyTypeFloat PropertyType = "float"

	// PropertyTypeDate is an ISO 8601 date (e.g. "2026-01-15").
	PropertyTypeDate PropertyType = "date"

	// PropertyTypeDatetime is an ISO 8601 date + time (e.g. "2026-01-15T10:30:00Z").
	PropertyTypeDatetime PropertyType = "datetime"

	// PropertyTypeBoolean is a true/false flag.
	PropertyTypeBoolean PropertyType = "boolean"

	// PropertyTypeUUID is an immutable RFC 4122 UUID string (e.g. "550e8400-e29b-41d4-a716-446655440000").
	// System-assigned at entity creation time; never set by user input.
	PropertyTypeUUID PropertyType = "uuid"

	// Choice types — allowed values are stored as runtime data, not in the schema.

	// PropertyTypeOption is a single fixed value from a predefined set.
	PropertyTypeOption PropertyType = "option"

	// PropertyTypeSelect is a single value chosen from a list at runtime.
	PropertyTypeSelect PropertyType = "select"

	// PropertyTypeMultiSelect is one or more values chosen from a list at runtime.
	PropertyTypeMultiSelect PropertyType = "multiselect"

	// Complex types — require additional configuration on [PropertyDefinition].

	// PropertyTypeRating is a numeric rating with a configurable range and labels.
	// A non-nil [RatingConfig] must be supplied on the [PropertyDefinition].
	PropertyTypeRating PropertyType = "rating"
)

// RatingConfig holds the configuration for a property of type [PropertyTypeRating].
// The Agency Owner provides Min, Max, and optionally Labels when defining the property.
type RatingConfig struct {
	// Min is the lowest allowed rating value (e.g. 1).
	Min int

	// Max is the highest allowed rating value (e.g. 5).
	Max int

	// Labels are optional human-readable names for each value from Min to Max.
	// If provided, len(Labels) must equal Max - Min + 1.
	Labels []string
}

// PropertyDefinition describes a single named property within a [TypeDefinition].
type PropertyDefinition struct {
	// Name is the property identifier (e.g. "pressure", "status", "rating").
	Name string

	// Type is the data type for this property.
	Type PropertyType

	// Required indicates that every instance of this type must supply this property.
	Required bool

	// RatingConfig holds the configuration for rating properties.
	// Must be non-nil when Type is [PropertyTypeRating]; ignored for all other types.
	RatingConfig *RatingConfig
}

// RelationshipDefinition declares a legal directed edge from the owning
// [TypeDefinition] to another type within the same [Schema].
//
// In OWL terms this is an ObjectProperty with a fixed rdfs:domain (the owning
// TypeDefinition) and rdfs:range (ToType). At the storage level each
// RelationshipDefinition corresponds to an ArangoDB edge label within the
// service's named graph.
//
// Validation rules applied by [DataManager.CreateRelationship]:
//   - The edge label must match a RelationshipDefinition.Name on the source entity's TypeDefinition.
//   - The target entity's TypeID must equal ToType.
//   - If ToMany is false, a second edge with the same label from the same source returns ErrRelationshipCardinalityViolation.
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

	// ToMany controls cardinality.
	//   false → at most one target entity (functional; owl:maxCardinality 1)
	//   true  → zero or more targets (collection; unbounded)
	ToMany bool

	// Required indicates that at least one edge of this label must exist on
	// every entity of the owning type (owl:minCardinality 1 when true).
	Required bool

	// Inverse is the optional name of the reciprocal relationship label on the
	// ToType (e.g. "belongs_to_agency"). If set, DataManager implementations
	// may use this to auto-create the inverse edge; behaviour is
	// implementation-defined and not enforced by the schema layer.
	Inverse string

	// PathSegment is the URL sub-resource segment used in schema-driven HTTP
	// route generation (e.g. "workflows" produces
	// /v{ver}/{agencyID}/{typeSeg}/{id}/workflows).
	// Must be lowercase, hyphen-separated, and unique within the owning TypeDefinition.
	// If empty, no sub-resource routes are generated for this relationship.
	PathSegment string
}

// TypeDefinition declares a named class of entity within a [Schema].
//
// In the Digital Twin service a type is a real-world entity class
// (e.g. "Pump", "Pipe", "Person"). In the Comm service a type is a participant
// or message class (e.g. "Channel", "Subscriber"). The same structure is used
// in both services; semantics differ by context.
type TypeDefinition struct {
	// Name is the unique type identifier within the Schema (e.g. "Pump").
	Name string

	// DisplayName is a human-readable label for this type (e.g. "Water Pump").
	DisplayName string

	// PathSegment is the URL segment used in schema-driven HTTP route generation
	// (e.g. "goal-templates" produces /v{ver}/{agencyID}/goal-templates).
	// Must be lowercase, hyphen-separated, and unique within the schema.
	// If empty, the type is not represented in the generated route set.
	PathSegment string

	// Properties is the ordered list of property definitions for this type.
	Properties []PropertyDefinition

	// Relationships is the ordered list of relationship definitions for this
	// type. Each entry declares a legal directed edge (ObjectProperty) this
	// type may form to another type in the same Schema.
	// An empty slice means this type has no declared outbound relationships;
	// the DataManager will reject any CreateRelationship call whose label is
	// not listed here.
	Relationships []RelationshipDefinition

	// StorageCollection is the backing ArangoDB collection for instances of this
	// type. If empty, the service default is used (e.g. "dt_entities" for
	// CodeValdDT). Set to "dt_telemetry" or "dt_events" to route writes to a
	// specialised collection.
	StorageCollection string

	// Immutable indicates that instances of this type cannot be updated after
	// creation. UpdateEntity returns [ErrImmutableType] when called on an entity
	// whose TypeDefinition has Immutable set to true. Only CreateEntity and
	// DeleteEntity are valid for immutable types.
	Immutable bool

	// EntityIDParam is the URL placeholder name used for the entity-ID segment in
	// schema-driven HTTP route generation (e.g. "workflowId" produces
	// /agency/{agencyId}/workflows/{workflowId}).
	// When empty, schemaroutes.RoutesFromSchema skips per-entity and
	// relationship routes for this type — only the collection-level list and
	// create routes are emitted.
	EntityIDParam string

	// UniqueKey is the ordered list of property names that together form a
	// composite natural key for this type (e.g. ["Code"] or ["Code", "ParentID"]).
	// When set, DataManager.UpsertEntity uses these property values to locate an
	// existing non-deleted entity and merge the supplied properties onto it,
	// instead of inserting a duplicate.
	// All names must reference a PropertyDefinition.Name declared in Properties.
	// An empty or nil slice means no unique key is defined — UpsertEntity returns
	// ErrUniqueKeyNotDefined for this type.
	UniqueKey []string

	// Code is the human-readable, user-facing label for this TypeDefinition
	// (e.g. "draft_work_item"). Unlike Name, which is the schema-internal
	// PascalCase key used in code, Code is the stable snake_case identifier
	// surfaced in API responses, UI labels, and external integrations.
	// It may be revised when the schema is updated but should be treated as
	// stable once deployed.
	Code string

	// RefCode is a pre-assigned, immutable UUID that serves as the stable
	// cross-reference identifier for this TypeDefinition. Unlike code, which
	// is a user-facing, human-readable label that may be renamed, RefCode is
	// generated once before deployment (via the generate_ref_codes.py script)
	// and never changes. It is the canonical key used in cross-entity reference
	// properties (e.g. draft_workflow_ref_code) and in any external system that
	// needs to address this type by a stable, opaque handle.
	// Must be a valid UUID v4 string (e.g. "a1b2c3d4-e5f6-7890-abcd-ef1234567890").
	RefCode string
}

// Schema is a versioned, immutable collection of [TypeDefinition]s for one
// service within one agency. Each service (DT, Comm) maintains its own
// independent Schema per agency. Updating the schema produces a new version;
// previous versions are preserved.
type Schema struct {
	// ID is the unique identifier for this schema version (UUID).
	ID string

	// AgencyID is the agency this schema belongs to.
	AgencyID string

	// Version is the auto-incrementing version number (1, 2, 3, …).
	// The first publish produces Version 1; each subsequent call increments by one.
	// Draft documents always carry Version 0.
	Version int

	// Active is true for the single published schema version that is currently
	// in use for write operations (CreateEntity, CreateRelationship).
	// Only one published version per agency can be active at a time.
	// Draft documents always have Active = false.
	Active bool

	// Tag is the human-readable version label (e.g. "v1", "v2").
	Tag string

	// Types is the ordered list of type definitions in this schema version.
	Types []TypeDefinition

	// CreatedAt is the time this schema version was created.
	CreatedAt time.Time
}
