// Package entitygraph provides the generic DataManager and SchemaManager
// interfaces, together with all associated request, filter, and result types,
// for any CodeVald service that owns a typed, graph-structured entity store
// backed by a versioned schema.
//
// Currently consumed by CodeValdDT (digital-twin graphs) and CodeValdComm
// (messaging/notification graphs). Both services alias these interfaces locally
// and supply their own ArangoDB-backed implementations via constructor
// injection. CodeValdAgency uses the same infrastructure for its entity store.
//
// Schema types (Schema, TypeDefinition, PropertyDefinition, PropertyType, …)
// are defined in the sibling [github.com/aosanya/CodeValdSharedLib/types]
// package and imported here by reference.
package entitygraph

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aosanya/CodeValdSharedLib/types"
)

// ErrInvalidRelationship is returned by CreateRelationship when the edge label
// is not declared in the source TypeDefinition.Relationships, or when the
// target entity's TypeID does not match RelationshipDefinition.ToType.
var ErrInvalidRelationship = errors.New("invalid relationship")

// ErrRelationshipCardinalityViolation is returned by CreateRelationship when
// a second edge with the same label is created from the same source entity and
// RelationshipDefinition.ToMany is false (functional / at-most-one).
var ErrRelationshipCardinalityViolation = errors.New("relationship cardinality violation")

// ErrRequiredRelationshipViolation is returned when an operation (e.g.
// DeleteEntity) would leave an entity without at least one edge for a
// relationship declared with Required = true.
var ErrRequiredRelationshipViolation = errors.New("required relationship violation")

// ErrSchemaNotFound is returned by SchemaManager methods when no schema
// document (draft or published) exists for the given agency or version.
var ErrSchemaNotFound = errors.New("schema not found")

// ErrEntityNotFound is returned by GetEntity, UpdateEntity, DeleteEntity, and
// CreateRelationship when the referenced entity does not exist.
var ErrEntityNotFound = errors.New("entity not found")

// ErrEntityAlreadyExists is returned by CreateEntity when an entity with the
// same ID already exists for the agency.
var ErrEntityAlreadyExists = errors.New("entity already exists")

// ErrRelationshipNotFound is returned by GetRelationship and DeleteRelationship
// when no relationship with the given ID exists for the agency.
var ErrRelationshipNotFound = errors.New("relationship not found")

// ErrImmutableType is returned by UpdateEntity when the entity's TypeDefinition
// has Immutable set to true.
var ErrImmutableType = errors.New("entity type is immutable")

// DataManager is the business-logic entry point for entity lifecycle and graph
// operations. Consumers alias this as their own service-scoped interface
// (e.g. DTDataManager = entitygraph.DataManager).
//
// Schema operations are not in scope — see SchemaManager.
//
// Immutable types (TypeDefinition.Immutable == true) reject UpdateEntity with
// ErrImmutableType. Storage routing is driven by TypeDefinition.StorageCollection.
//
// All methods accept context.Context as the first argument for cancellation and
// deadline propagation.
type DataManager interface {
	// CreateEntity creates a new entity of the given type for the agency.
	// The TypeID must match a TypeDefinition.Name in the agency's current schema.
	// Returns ErrEntityAlreadyExists if an entity with the same ID already exists.
	CreateEntity(ctx context.Context, req CreateEntityRequest) (Entity, error)

	// GetEntity returns the entity identified by agencyID and entityID.
	// Returns ErrEntityNotFound if no entity matches.
	GetEntity(ctx context.Context, agencyID, entityID string) (Entity, error)

	// UpdateEntity patches the properties of an existing entity.
	// Returns ErrEntityNotFound if the entity does not exist.
	// Returns ErrImmutableType if the entity's type has Immutable set to true.
	UpdateEntity(ctx context.Context, agencyID, entityID string, req UpdateEntityRequest) (Entity, error)

	// DeleteEntity soft-deletes the entity by setting Deleted=true and
	// recording DeletedAt. The entity is never hard-deleted in v1.
	// Relationships referencing the entity are retained as orphans.
	// Returns ErrEntityNotFound if the entity does not exist.
	DeleteEntity(ctx context.Context, agencyID, entityID string) error

	// ListEntities returns all entities matching the filter.
	// Soft-deleted entities are excluded from the results.
	ListEntities(ctx context.Context, filter EntityFilter) ([]Entity, error)

	// CreateRelationship creates a directed edge between two entities.
	// Returns ErrEntityNotFound if either the FromID or ToID entity does not exist.
	CreateRelationship(ctx context.Context, req CreateRelationshipRequest) (Relationship, error)

	// GetRelationship returns the relationship identified by agencyID and
	// relationshipID.
	// Returns ErrRelationshipNotFound if no relationship matches.
	GetRelationship(ctx context.Context, agencyID, relationshipID string) (Relationship, error)

	// DeleteRelationship removes the edge permanently.
	// Returns ErrRelationshipNotFound if no relationship matches.
	DeleteRelationship(ctx context.Context, agencyID, relationshipID string) error

	// ListRelationships returns all edges matching the filter.
	// Zero-value filter fields are ignored (no filtering on that field).
	ListRelationships(ctx context.Context, filter RelationshipFilter) ([]Relationship, error)

	// TraverseGraph walks the entity graph from StartID to the given Depth and
	// returns all reachable vertices and traversed edges.
	// Soft-deleted entities are excluded from the result vertices.
	TraverseGraph(ctx context.Context, req TraverseGraphRequest) (TraverseGraphResult, error)
}

// SchemaManager is the schema storage contract injected into a concrete
// DataManager implementation. It separates the mutable draft schema
// (one document per agency, overwritten by SetSchema) from the immutable
// published history (append-only snapshots produced by Publish and Activate).
//
// Updating the draft does not affect live traffic — callers must Publish and
// then Activate a version before it is used by CreateEntity / CreateRelationship.
type SchemaManager interface {
	// Draft collection — one mutable document per agency.

	// SetSchema overwrites the agency's current draft schema.
	// The draft is never versioned; only published snapshots carry version numbers.
	// ValidateSchema is NOT called here — invalid drafts are permitted until Publish.
	SetSchema(ctx context.Context, schema types.Schema) error

	// GetSchema returns the agency's current draft schema.
	// Returns ErrSchemaNotFound if no draft has been created yet.
	GetSchema(ctx context.Context, agencyID string) (types.Schema, error)

	// Published collection — immutable, append-only.

	// Publish validates the current draft (ValidateSchema) and snapshots it into
	// the published collection as a new version with Active = false.
	// The version number is auto-assigned (highest existing + 1; first publish = 1).
	// Returns an error and creates no snapshot if validation fails or no draft exists.
	Publish(ctx context.Context, agencyID string) error

	// Activate promotes the given published version to active, setting Active = true
	// on the target and Active = false on any previously active version, in a single
	// transaction. Returns ErrSchemaNotFound if the version does not exist.
	Activate(ctx context.Context, agencyID string, version int) error

	// GetActive returns the single published version where Active == true.
	// Returns ErrSchemaNotFound if no version has been activated yet.
	GetActive(ctx context.Context, agencyID string) (types.Schema, error)

	// GetVersion returns a specific published version.
	// Returns ErrSchemaNotFound if the version does not exist.
	GetVersion(ctx context.Context, agencyID string, version int) (types.Schema, error)

	// ListVersions returns all published versions for the agency in ascending
	// version order. Includes both active and inactive versions.
	ListVersions(ctx context.Context, agencyID string) ([]types.Schema, error)
}

// Entity is an instance of a typed real-world object managed by a DataManager.
// TypeID matches TypeDefinition.Name in the agency's current schema.
// Properties hold the current state values; no schema validation is performed
// in v1.
// Deleted and DeletedAt are set by DeleteEntity (soft delete) — the entity is
// never hard-deleted in v1.
type Entity struct {
	// ID is the unique identifier for this entity (UUID).
	ID string

	// AgencyID is the agency this entity belongs to.
	AgencyID string

	// TypeID matches TypeDefinition.Name in the agency's current schema
	// (e.g. "Pump", "Channel").
	TypeID string

	// Properties holds the current state values keyed by property name.
	// No schema validation is applied in v1.
	Properties map[string]any

	// CreatedAt is the time this entity was created.
	CreatedAt time.Time

	// UpdatedAt is the time this entity was last updated.
	UpdatedAt time.Time

	// Deleted is true once DeleteEntity has been called.
	Deleted bool

	// DeletedAt is set when DeleteEntity is called; nil until then.
	DeletedAt *time.Time
}

// CreateEntityRequest is the input for creating a new entity.
type CreateEntityRequest struct {
	// AgencyID is the owning agency.
	AgencyID string

	// TypeID must match a TypeDefinition.Name in the agency's current schema.
	TypeID string

	// Properties are the initial state values for the entity.
	Properties map[string]any

	// Relationships are optional inline edges to create atomically with the
	// entity. Each entry is validated against the TypeDefinition before any
	// writes are made. If any validation fails the entire operation is aborted.
	// Required relationships (RelationshipDefinition.Required == true) must be
	// supplied here; omitting them causes ErrRequiredRelationshipViolation.
	Relationships []EntityRelationshipRequest
}

// EntityRelationshipRequest carries a single relationship to create alongside
// a new entity in [CreateEntityRequest].
type EntityRelationshipRequest struct {
	// Name is the edge label — must match a RelationshipDefinition.Name declared
	// on the entity's TypeDefinition.
	Name string

	// ToID is the target entity ID.
	ToID string
}

// UpdateEntityRequest is the input for patching an entity's properties.
// Only the keys present in Properties are updated; absent keys are left unchanged.
type UpdateEntityRequest struct {
	// Properties are the property values to patch onto the entity.
	Properties map[string]any
}

// EntityFilter scopes a ListEntities query.
// Zero-value fields are ignored (no filtering applied for that field).
type EntityFilter struct {
	// AgencyID restricts results to this agency. If empty, all agencies are included.
	AgencyID string

	// TypeID restricts results to entities of this type.
	// If empty, all entity types are included.
	TypeID string
}

// Relationship is a directed graph edge between two entities.
// Stored in an ArangoDB edge collection — _from and _to reference entity
// documents.
type Relationship struct {
	// ID is the unique identifier for this relationship (UUID).
	ID string

	// AgencyID is the agency this relationship belongs to.
	AgencyID string

	// Name is the semantic label for this edge (e.g. "connects_to",
	// "reports_to").
	Name string

	// FromID is the source entity ID.
	FromID string

	// ToID is the target entity ID.
	ToID string

	// Properties are optional metadata carried on the edge.
	Properties map[string]any

	// CreatedAt is the time this relationship was created.
	CreatedAt time.Time
}

// CreateRelationshipRequest is the input for creating a directed graph edge
// between two entities.
type CreateRelationshipRequest struct {
	// AgencyID is the owning agency.
	AgencyID string

	// Name is the semantic label for the edge.
	Name string

	// FromID is the source entity ID.
	FromID string

	// ToID is the target entity ID.
	ToID string

	// Properties are optional metadata to store on the edge.
	Properties map[string]any
}

// RelationshipFilter scopes a ListRelationships query.
// Zero-value fields are ignored (no filtering applied for that field).
type RelationshipFilter struct {
	// AgencyID restricts results to this agency.
	AgencyID string

	// FromID filters by source entity ID; empty means any source.
	FromID string

	// ToID filters by target entity ID; empty means any target.
	ToID string

	// Name filters by relationship type label; empty means all labels.
	Name string
}

// TraverseGraphRequest walks the entity graph from a starting entity.
type TraverseGraphRequest struct {
	// AgencyID is the owning agency.
	AgencyID string

	// StartID is the entity ID from which traversal begins.
	StartID string

	// Direction is the traversal direction: "outbound", "inbound", or "any".
	Direction string

	// Depth is the maximum traversal depth. 0 is treated as 1.
	Depth int

	// Names restricts traversal to edges whose Name is in this list.
	// An empty or nil slice means no filtering — all reachable edges are followed
	// regardless of label.
	Names []string
}

// TraverseGraphResult is returned by TraverseGraph.
// Both visited vertices and traversed edges are included so callers can inspect
// relationship names and properties without a second round-trip.
type TraverseGraphResult struct {
	// Vertices are all reachable entities, excluding soft-deleted ones.
	Vertices []Entity

	// Edges are the traversed relationships in order of discovery.
	Edges []Relationship
}

// FindTypeDef returns the [types.TypeDefinition] for the given typeName within
// the schema, or an error if no matching type exists.
//
// This helper is intended for use by DataManager implementations that need to
// look up the definition of an entity's type before executing a write
// operation (e.g. to retrieve its StorageCollection or its declared
// Relationships for validation).
func FindTypeDef(schema types.Schema, typeName string) (types.TypeDefinition, error) {
	for _, td := range schema.Types {
		if td.Name == typeName {
			return td, nil
		}
	}
	return types.TypeDefinition{}, fmt.Errorf("type %q not found in schema %s", typeName, schema.ID)
}

// FindRelationshipDef returns the [types.RelationshipDefinition] with the
// given label declared on the source TypeDefinition, or an error if no
// matching definition exists.
//
// Implementations should call this inside CreateRelationship to validate the
// edge label and target type before writing to the edge collection.
//
//	td, err := entitygraph.FindTypeDef(schema, fromEntity.TypeID)
//	if err != nil { return entitygraph.ErrInvalidRelationship }
//	rd, err := entitygraph.FindRelationshipDef(td, label)
//	if err != nil { return entitygraph.ErrInvalidRelationship }
//	if rd.ToType != toEntity.TypeID { return entitygraph.ErrInvalidRelationship }
func FindRelationshipDef(td types.TypeDefinition, label string) (types.RelationshipDefinition, error) {
	for _, rd := range td.Relationships {
		if rd.Name == label {
			return rd, nil
		}
	}
	return types.RelationshipDefinition{}, fmt.Errorf("relationship %q not declared on type %q", label, td.Name)
}

// ValidateCreateRelationship checks that the proposed edge is permitted by the
// schema. Must be called by every DataManager backend before writing an edge.
//
// Rules enforced:
//  1. label must match a RelationshipDefinition.Name on fromTypeDef.
//  2. toTypeID must equal RelationshipDefinition.ToType.
//
// Cardinality (ToMany=false upsert vs. ToMany=true insert) is handled by the
// backend write strategy — not by this function.
//
// Returns [ErrInvalidRelationship] if either rule is violated.
func ValidateCreateRelationship(fromTypeDef types.TypeDefinition, label, toTypeID string) error {
	rd, err := FindRelationshipDef(fromTypeDef, label)
	if err != nil {
		return ErrInvalidRelationship
	}
	if rd.ToType != toTypeID {
		return ErrInvalidRelationship
	}
	return nil
}

// ValidateSchema checks the internal consistency of a [types.Schema] before it
// is persisted by [SchemaManager.Publish]. Called inside Publish — invalid
// schemas are rejected and no snapshot is created.
//
// Rules enforced:
//  1. All TypeDefinition.Name values are unique within the schema.
//  2. All TypeDefinition.PathSegment values are unique within the schema
//     (non-empty segments only).
//  3. For every RelationshipDefinition where Inverse != "":
//     a. ToType must reference a TypeDefinition.Name in the same schema.
//     b. The ToType's TypeDefinition must declare a RelationshipDefinition
//     with Name == rd.Inverse.
//  4. Within each TypeDefinition, all RelationshipDefinition.PathSegment
//     values are unique (non-empty segments only).
//
// Returns a descriptive error on the first violation found.
func ValidateSchema(schema types.Schema) error {
	typeNames := make(map[string]struct{}, len(schema.Types))
	typePathSegs := make(map[string]struct{}, len(schema.Types))

	for _, td := range schema.Types {
		if _, dup := typeNames[td.Name]; dup {
			return fmt.Errorf("ValidateSchema %s: duplicate type name %q", schema.AgencyID, td.Name)
		}
		typeNames[td.Name] = struct{}{}

		if td.PathSegment != "" {
			if _, dup := typePathSegs[td.PathSegment]; dup {
				return fmt.Errorf("ValidateSchema %s: duplicate type PathSegment %q", schema.AgencyID, td.PathSegment)
			}
			typePathSegs[td.PathSegment] = struct{}{}
		}
	}

	for _, td := range schema.Types {
		relPathSegs := make(map[string]struct{}, len(td.Relationships))
		for _, rd := range td.Relationships {
			if rd.Inverse != "" {
				toTypeDef, err := FindTypeDef(schema, rd.ToType)
				if err != nil {
					return fmt.Errorf("ValidateSchema %s: type %q: relationship %q: ToType %q not found in schema",
						schema.AgencyID, td.Name, rd.Name, rd.ToType)
				}
				if _, err := FindRelationshipDef(toTypeDef, rd.Inverse); err != nil {
					return fmt.Errorf("ValidateSchema %s: type %q: relationship %q: inverse %q not declared on %q",
						schema.AgencyID, td.Name, rd.Name, rd.Inverse, rd.ToType)
				}
			}
			if rd.PathSegment != "" {
				if _, dup := relPathSegs[rd.PathSegment]; dup {
					return fmt.Errorf("ValidateSchema %s: type %q: duplicate relationship PathSegment %q",
						schema.AgencyID, td.Name, rd.PathSegment)
				}
				relPathSegs[rd.PathSegment] = struct{}{}
			}
		}
	}

	return nil
}
