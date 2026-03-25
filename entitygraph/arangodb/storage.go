// Package arangodb provides a generic ArangoDB implementation of
// [entitygraph.DataManager] and [entitygraph.SchemaManager] for CodeVald
// services that own a typed, graph-structured entity store.
//
// All service-specific names (collection names, graph name, default database)
// are supplied via [Config] — nothing is hardcoded. Each consuming service
// wraps this package in a thin service-scoped adapter that fills in the
// appropriate names.
//
// Infrastructure collections managed by this package:
//   - EntityCollection    — document collection; fallback for TypeIDs without a StorageCollection
//   - RelCollection       — ArangoDB edge collection for directed graph edges
//   - SchemasDraftCol     — one mutable document per agency (draft schema)
//   - SchemasPublishedCol — immutable append-only published schema snapshots
//
// File layout:
//   - storage.go       — Config, Backend struct, constructors, collection setup
//   - entities.go      — CreateEntity, GetEntity, UpdateEntity, DeleteEntity, ListEntities, UpsertEntity
//   - relationships.go — CreateRelationship, GetRelationship, DeleteRelationship,
//     ListRelationships, TraverseGraph
//   - schemaops.go     — SetSchema, GetSchema, Publish, Activate, GetActive,
//     GetVersion, ListVersions
//
// Use [New] to obtain a (DataManager, SchemaManager) pair from an open database.
// Use [NewBackend] to connect and construct in a single call.
// Use [NewBackendFromDB] in tests that manage their own database lifecycle.
package arangodb

import (
	"context"
	"fmt"
	"time"

	driver "github.com/arangodb/go-driver"

	"github.com/aosanya/CodeValdSharedLib/arangoutil"
	"github.com/aosanya/CodeValdSharedLib/entitygraph"
	"github.com/aosanya/CodeValdSharedLib/types"
)

// ConnConfig holds the connection parameters common to every CodeVald service
// that uses this package. Service-specific wrappers declare:
//
//	type Config = sharedadb.ConnConfig
//
// and then expand it to a full [Config] by filling in the collection and
// graph names via a service-local toSharedConfig function.
type ConnConfig struct {
	// Endpoint is the ArangoDB HTTP endpoint (e.g. "http://localhost:8529").
	// Defaults to "http://localhost:8529" when empty.
	Endpoint string

	// Username is the ArangoDB username. Defaults to "root" when empty.
	Username string

	// Password is the ArangoDB password.
	Password string

	// Database is the ArangoDB database name.
	// Each service supplies its own default (e.g. "codevaldagency", "codevaldai").
	Database string

	// Schema drives which entity collections are created. Collection names and
	// immutability are derived from TypeDefinition.StorageCollection and
	// TypeDefinition.Immutable.
	Schema types.Schema
}

// Config holds connection parameters and collection/graph names for the
// ArangoDB backend. All collection and graph name fields are required.
type Config struct {
	// Endpoint is the ArangoDB HTTP endpoint (e.g. "http://localhost:8529").
	// Defaults to "http://localhost:8529" when empty.
	Endpoint string

	// Username is the ArangoDB username. Defaults to "root" when empty.
	Username string

	// Password is the ArangoDB password.
	Password string

	// Database is the ArangoDB database name. Required — each service supplies
	// its own default (e.g. "codevaldagency", "codevaldai").
	Database string

	// Schema drives which entity collections are created. Collection names and
	// immutability are derived from TypeDefinition.StorageCollection and
	// TypeDefinition.Immutable.
	Schema types.Schema

	// EntityCollection is the fallback document collection used for TypeIDs
	// that do not declare a StorageCollection in the schema
	// (e.g. "agency_entities", "ai_entities").
	EntityCollection string

	// RelCollection is the ArangoDB edge collection for directed graph edges
	// (e.g. "agency_relationships", "ai_relationships").
	RelCollection string

	// SchemasDraftCol is the collection for mutable draft schema documents,
	// one per agency (e.g. "agency_schemas_draft", "ai_schemas_draft").
	SchemasDraftCol string

	// SchemasPublishedCol is the collection for immutable published schema
	// snapshots (e.g. "agency_schemas_published", "ai_schemas_published").
	SchemasPublishedCol string

	// GraphName is the ArangoDB named graph
	// (e.g. "agency_graph", "ai_graph").
	GraphName string
}

// Backend is the ArangoDB implementation of both [entitygraph.DataManager] and
// [entitygraph.SchemaManager]. It is obtained via [New], [NewBackend], or
// [NewBackendFromDB].
//
// entityColMap maps collection name → driver.Collection for every collection
// referenced by the schema TypeDefinitions plus the fallback EntityCollection.
// typeDefs maps TypeID → TypeDefinition for O(1) immutability and
// StorageCollection lookups.
//
// The name fields (relCollectionName, graphName, schemasDraftName,
// schemasPublishedName) are stored for use in AQL query strings.
type Backend struct {
	db                   driver.Database
	entityColMap         map[string]driver.Collection    // collection name → driver.Collection
	typeDefs             map[string]types.TypeDefinition // TypeID → TypeDefinition
	fallback             driver.Collection               // EntityCollection
	relationships        driver.Collection
	schemasDraft         driver.Collection
	schemasPublished     driver.Collection
	relCollectionName    string // used in ListRelationships AQL
	graphName            string // used in TraverseGraph AQL
	schemasDraftName     string // used in schemaops AQL
	schemasPublishedName string // used in schemaops AQL
}

// collectionFor returns the driver.Collection for the given TypeID,
// falling back to the EntityCollection when StorageCollection is empty.
func (b *Backend) collectionFor(typeID string) driver.Collection {
	if td, ok := b.typeDefs[typeID]; ok && td.StorageCollection != "" {
		if col, ok := b.entityColMap[td.StorageCollection]; ok {
			return col
		}
	}
	return b.fallback
}

// isImmutable returns true when the TypeDefinition for typeID has Immutable set.
func (b *Backend) isImmutable(typeID string) bool {
	if td, ok := b.typeDefs[typeID]; ok {
		return td.Immutable
	}
	return false
}

// allEntityCollections returns every distinct entity collection derived from
// the schema — used for graph vertex lists and cross-collection searches.
func (b *Backend) allEntityCollections() []driver.Collection {
	seen := make(map[string]struct{})
	var cols []driver.Collection
	for _, td := range b.typeDefs {
		name := td.StorageCollection
		if name == "" {
			name = b.fallback.Name()
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		if col, ok := b.entityColMap[name]; ok {
			cols = append(cols, col)
		}
	}
	// Always include the fallback even if no TypeID maps to it explicitly.
	if _, ok := seen[b.fallback.Name()]; !ok {
		cols = append(cols, b.fallback)
	}
	return cols
}

// New constructs a Backend from an already-open driver.Database using the
// provided Config, ensures all collections and the named graph exist, and
// returns the Backend as both a DataManager and a SchemaManager.
func New(db driver.Database, cfg Config) (entitygraph.DataManager, entitygraph.SchemaManager, error) {
	b, err := newBackendFromDB(context.Background(), db, cfg)
	if err != nil {
		return nil, nil, err
	}
	return b, b, nil
}

// NewBackend connects to ArangoDB using cfg, ensures all collections exist,
// and returns a ready-to-use Backend. cfg.Schema drives which entity
// collections are created. cfg.Database, cfg.EntityCollection, cfg.RelCollection,
// cfg.SchemasDraftCol, cfg.SchemasPublishedCol, and cfg.GraphName are required.
func NewBackend(cfg Config) (*Backend, error) {
	if cfg.Database == "" {
		return nil, fmt.Errorf("arangodb: NewBackend: Database must be set")
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://localhost:8529"
	}
	if cfg.Username == "" {
		cfg.Username = "root"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := arangoutil.Connect(ctx, arangoutil.Config{
		Endpoint: cfg.Endpoint,
		Username: cfg.Username,
		Password: cfg.Password,
		Database: cfg.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("arangodb: %w", err)
	}

	return newBackendFromDB(ctx, db, cfg)
}

// NewBackendFromDB constructs a Backend from an already-open driver.Database
// using the provided Config. Intended for tests that manage their own database
// lifecycle.
func NewBackendFromDB(db driver.Database, cfg Config) (*Backend, error) {
	if db == nil {
		return nil, fmt.Errorf("arangodb: NewBackendFromDB: database must not be nil")
	}
	return newBackendFromDB(context.Background(), db, cfg)
}

func newBackendFromDB(ctx context.Context, db driver.Database, cfg Config) (*Backend, error) {
	// Build typeDefs index for O(1) lookups.
	typeDefs := make(map[string]types.TypeDefinition, len(cfg.Schema.Types))
	for _, td := range cfg.Schema.Types {
		typeDefs[td.Name] = td
	}

	// Collect unique StorageCollection names from the schema.
	colNames := make(map[string]struct{})
	for _, td := range cfg.Schema.Types {
		if td.StorageCollection != "" {
			colNames[td.StorageCollection] = struct{}{}
		}
	}
	// Always ensure the fallback collection exists even if nothing maps to it.
	colNames[cfg.EntityCollection] = struct{}{}

	// Ensure every entity collection exists in ArangoDB.
	entityColMap := make(map[string]driver.Collection, len(colNames))
	for name := range colNames {
		col, err := ensureDocumentCollection(ctx, db, name)
		if err != nil {
			return nil, fmt.Errorf("ensure entity collection %q: %w", name, err)
		}
		entityColMap[name] = col
	}

	// Ensure infrastructure collections.
	relationships, err := ensureEdgeCollection(ctx, db, cfg.RelCollection)
	if err != nil {
		return nil, fmt.Errorf("ensure %q: %w", cfg.RelCollection, err)
	}
	schemasDraft, err := ensureDocumentCollection(ctx, db, cfg.SchemasDraftCol)
	if err != nil {
		return nil, fmt.Errorf("ensure %q: %w", cfg.SchemasDraftCol, err)
	}
	schemasPublished, err := ensureDocumentCollection(ctx, db, cfg.SchemasPublishedCol)
	if err != nil {
		return nil, fmt.Errorf("ensure %q: %w", cfg.SchemasPublishedCol, err)
	}

	b := &Backend{
		db:                   db,
		entityColMap:         entityColMap,
		typeDefs:             typeDefs,
		fallback:             entityColMap[cfg.EntityCollection],
		relationships:        relationships,
		schemasDraft:         schemasDraft,
		schemasPublished:     schemasPublished,
		relCollectionName:    cfg.RelCollection,
		graphName:            cfg.GraphName,
		schemasDraftName:     cfg.SchemasDraftCol,
		schemasPublishedName: cfg.SchemasPublishedCol,
	}

	if err := ensureGraph(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

// ensureGraph creates the named ArangoDB graph if it does not already exist.
// Vertex collections are derived from the backend schema.
func ensureGraph(ctx context.Context, b *Backend) error {
	exists, err := b.db.GraphExists(ctx, b.graphName)
	if err != nil {
		return fmt.Errorf("ensureGraph: check exists: %w", err)
	}
	if exists {
		return nil
	}
	all := b.allEntityCollections()
	names := make([]string, len(all))
	for i, col := range all {
		names[i] = col.Name()
	}
	_, err = b.db.CreateGraph(ctx, b.graphName, &driver.CreateGraphOptions{
		EdgeDefinitions: []driver.EdgeDefinition{
			{
				Collection: b.relCollectionName,
				From:       names,
				To:         names,
			},
		},
	})
	if err != nil && !driver.IsConflict(err) {
		return fmt.Errorf("ensureGraph: create: %w", err)
	}
	return nil
}

func ensureDocumentCollection(ctx context.Context, db driver.Database, name string) (driver.Collection, error) {
	exists, err := db.CollectionExists(ctx, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return db.Collection(ctx, name)
	}
	col, err := db.CreateCollection(ctx, name, nil)
	if err != nil {
		if driver.IsConflict(err) {
			return db.Collection(ctx, name)
		}
		return nil, err
	}
	return col, nil
}

func ensureEdgeCollection(ctx context.Context, db driver.Database, name string) (driver.Collection, error) {
	exists, err := db.CollectionExists(ctx, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return db.Collection(ctx, name)
	}
	col, err := db.CreateCollection(ctx, name, &driver.CreateCollectionOptions{
		Type: driver.CollectionTypeEdge,
	})
	if err != nil {
		if driver.IsConflict(err) {
			return db.Collection(ctx, name)
		}
		return nil, err
	}
	return col, nil
}
