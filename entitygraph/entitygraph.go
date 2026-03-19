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
// DataManager implementation. It owns read and write access to the service's
// schema collection (e.g. dt_schemas for CodeValdDT, comm_schemas for
// CodeValdComm).
//
// Updating the schema produces a new immutable version; previous versions are
// preserved and readable via GetSchema.
type SchemaManager interface {
	// SetSchema stores a new schema version for the given agency.
	// The version number is set by the implementation (auto-increment from the
	// highest existing version).
	SetSchema(ctx context.Context, schema types.Schema) error

	// GetSchema returns the schema at the given version for the agency.
	// Returns ErrSchemaNotFound if no schema exists for that version.
	GetSchema(ctx context.Context, agencyID string, version int) (types.Schema, error)

	// ListSchemaVersions returns all known schema versions for the agency in
	// ascending version order.
	ListSchemaVersions(ctx context.Context, agencyID string) ([]types.Schema, error)
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
