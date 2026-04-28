// relationships.go contains CreateRelationship, GetRelationship,
// DeleteRelationship, ListRelationships, and TraverseGraph for the Backend.
package arangodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	driver "github.com/arangodb/go-driver"
	"github.com/google/uuid"

	"github.com/aosanya/CodeValdSharedLib/entitygraph"
)

// relationshipDoc is the ArangoDB edge-document representation of an
// [entitygraph.Relationship]. The _from and _to fields reference entity
// documents in their respective collections.
type relationshipDoc struct {
	Key        string         `json:"_key,omitempty"`
	From       string         `json:"_from"`
	To         string         `json:"_to"`
	Name       string         `json:"name"`
	AgencyID   string         `json:"agency_id"`
	Properties map[string]any `json:"properties"`
	CreatedAt  time.Time      `json:"created_at"`
}

// entityHandle returns the ArangoDB document handle for an entity ID, e.g.
// "ai_entities/<id>". It searches every entity collection derived from the
// schema so that the correct collection prefix is used in edge documents.
func (b *Backend) entityHandle(ctx context.Context, agencyID, entityID string) (string, error) {
	for _, col := range b.allEntityCollections() {
		var doc entityDoc
		if _, err := col.ReadDocument(ctx, entityID, &doc); err == nil {
			if doc.AgencyID == agencyID {
				return col.Name() + "/" + entityID, nil
			}
		} else if !driver.IsNotFound(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("entityHandle %s: %w", entityID, entitygraph.ErrEntityNotFound)
}

// CreateRelationship creates a directed edge in the relationships collection.
// Returns entitygraph.ErrEntityNotFound if the FromID or ToID entity does not exist.
func (b *Backend) CreateRelationship(
	ctx context.Context,
	req entitygraph.CreateRelationshipRequest,
) (entitygraph.Relationship, error) {
	fromHandle, err := b.entityHandle(ctx, req.AgencyID, req.FromID)
	if err != nil {
		return entitygraph.Relationship{}, fmt.Errorf("CreateRelationship from: %w", err)
	}
	toHandle, err := b.entityHandle(ctx, req.AgencyID, req.ToID)
	if err != nil {
		return entitygraph.Relationship{}, fmt.Errorf("CreateRelationship to: %w", err)
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	doc := relationshipDoc{
		Key:        id,
		From:       fromHandle,
		To:         toHandle,
		Name:       req.Name,
		AgencyID:   req.AgencyID,
		Properties: req.Properties,
		CreatedAt:  now,
	}
	if doc.Properties == nil {
		doc.Properties = make(map[string]any)
	}
	if _, err := b.relationships.CreateDocument(ctx, doc); err != nil {
		if driver.IsConflict(err) {
			return entitygraph.Relationship{}, fmt.Errorf("CreateRelationship: relationship already exists")
		}
		return entitygraph.Relationship{}, fmt.Errorf("CreateRelationship: %w", err)
	}
	return toRelationship(doc, id), nil
}

// GetRelationship returns the relationship identified by agencyID and
// relationshipID. Returns entitygraph.ErrRelationshipNotFound if absent.
func (b *Backend) GetRelationship(
	ctx context.Context,
	agencyID, relationshipID string,
) (entitygraph.Relationship, error) {
	var doc relationshipDoc
	if _, err := b.relationships.ReadDocument(ctx, relationshipID, &doc); err != nil {
		if driver.IsNotFound(err) {
			return entitygraph.Relationship{}, fmt.Errorf("GetRelationship %s: %w", relationshipID, entitygraph.ErrRelationshipNotFound)
		}
		return entitygraph.Relationship{}, fmt.Errorf("GetRelationship %s: %w", relationshipID, err)
	}
	if doc.AgencyID != agencyID {
		return entitygraph.Relationship{}, fmt.Errorf("GetRelationship %s: %w", relationshipID, entitygraph.ErrRelationshipNotFound)
	}
	return toRelationship(doc, relationshipID), nil
}

// DeleteRelationship removes an edge document permanently.
// Returns entitygraph.ErrRelationshipNotFound if the relationship does not exist.
func (b *Backend) DeleteRelationship(
	ctx context.Context,
	agencyID, relationshipID string,
) error {
	if _, err := b.GetRelationship(ctx, agencyID, relationshipID); err != nil {
		return fmt.Errorf("DeleteRelationship %s: %w", relationshipID, err)
	}
	if _, err := b.relationships.RemoveDocument(ctx, relationshipID); err != nil {
		if driver.IsNotFound(err) {
			return fmt.Errorf("DeleteRelationship %s: %w", relationshipID, entitygraph.ErrRelationshipNotFound)
		}
		return fmt.Errorf("DeleteRelationship %s: %w", relationshipID, err)
	}
	return nil
}

// ListRelationships returns all edges matching the filter.
// Zero-value filter fields are treated as "no restriction".
func (b *Backend) ListRelationships(
	ctx context.Context,
	filter entitygraph.RelationshipFilter,
) ([]entitygraph.Relationship, error) {
	bindVars := map[string]interface{}{}
	conditions := []string{"1==1"}
	if filter.AgencyID != "" {
		conditions = append(conditions, "doc.agency_id == @agencyID")
		bindVars["agencyID"] = filter.AgencyID
	}
	if filter.Name != "" {
		conditions = append(conditions, "doc.name == @name")
		bindVars["name"] = filter.Name
	}
	if filter.FromID != "" {
		conditions = append(conditions, "doc._from LIKE CONCAT('%/', @fromID)")
		bindVars["fromID"] = filter.FromID
	}
	if filter.ToID != "" {
		conditions = append(conditions, "doc._to LIKE CONCAT('%/', @toID)")
		bindVars["toID"] = filter.ToID
	}
	where := ""
	for i, c := range conditions {
		if i == 0 {
			where = c
		} else {
			where += " AND " + c
		}
	}
	q := fmt.Sprintf("FOR doc IN %s FILTER %s RETURN doc", b.relCollectionName, where)
	cursor, err := b.db.Query(ctx, q, bindVars)
	if err != nil {
		return nil, fmt.Errorf("ListRelationships: query: %w", err)
	}
	var results []entitygraph.Relationship
	var readErr error
	for cursor.HasMore() {
		var doc relationshipDoc
		meta, rErr := cursor.ReadDocument(ctx, &doc)
		if rErr != nil {
			readErr = fmt.Errorf("ListRelationships: read: %w", rErr)
			break
		}
		results = append(results, toRelationship(doc, meta.Key))
	}
	cursor.Close()
	if readErr != nil {
		return nil, readErr
	}
	return results, nil
}

// TraverseGraph walks the named graph from the start entity up to the
// requested depth and returns both the traversed edges and the reachable
// non-deleted vertices.
//
// Filters honoured:
//   - req.AgencyID — edges from other agencies are excluded (hard isolation).
//   - req.Names    — when non-empty, only edges whose Name is in the list
//     are returned. An empty/nil Names follows every reachable edge.
//
// Direction accepts the entitygraph.TraverseGraphRequest contract values
// "outbound" / "inbound" / "any" (case-insensitive). An empty Direction
// defaults to OUTBOUND. Vertices are deduplicated by entity ID.
func (b *Backend) TraverseGraph(
	ctx context.Context,
	req entitygraph.TraverseGraphRequest,
) (entitygraph.TraverseGraphResult, error) {
	startHandle, err := b.entityHandle(ctx, req.AgencyID, req.StartID)
	if err != nil {
		return entitygraph.TraverseGraphResult{}, fmt.Errorf("TraverseGraph start: %w", err)
	}
	direction, err := normalizeDirection(req.Direction)
	if err != nil {
		return entitygraph.TraverseGraphResult{}, err
	}
	depth := req.Depth
	if depth <= 0 {
		depth = 1
	}
	bindVars := map[string]interface{}{
		"startVertex": startHandle,
		"depth":       depth,
		"agencyID":    req.AgencyID,
	}
	nameFilter := ""
	if len(req.Names) > 0 {
		nameFilter = "FILTER e.name IN @names"
		bindVars["names"] = req.Names
	}
	// Direction is an AQL keyword and cannot be parameterised — it is
	// validated by normalizeDirection above before being inlined.
	q := fmt.Sprintf(
		`FOR v, e IN 1..@depth %s @startVertex GRAPH '%s'
		 FILTER e.agency_id == @agencyID
		 %s
		 FILTER v.deleted != true
		 RETURN { v: v, e: e }`,
		direction, b.graphName, nameFilter,
	)
	cursor, err := b.db.Query(ctx, q, bindVars)
	if err != nil {
		return entitygraph.TraverseGraphResult{}, fmt.Errorf("TraverseGraph: query: %w", err)
	}
	defer cursor.Close()

	var (
		vertices []entitygraph.Entity
		edges    []entitygraph.Relationship
		seenVtx  = map[string]struct{}{}
	)
	for cursor.HasMore() {
		var row struct {
			V entityDoc       `json:"v"`
			E relationshipDoc `json:"e"`
		}
		// We need both meta keys; ReadDocument only returns the row's outer
		// metadata, so the inner _key fields decode via JSON tags.
		if _, rErr := cursor.ReadDocument(ctx, &row); rErr != nil {
			return entitygraph.TraverseGraphResult{}, fmt.Errorf("TraverseGraph: read: %w", rErr)
		}
		if _, ok := seenVtx[row.V.Key]; !ok {
			seenVtx[row.V.Key] = struct{}{}
			vertices = append(vertices, toEntity(row.V, row.V.Key))
		}
		edges = append(edges, toRelationship(row.E, row.E.Key))
	}
	return entitygraph.TraverseGraphResult{Vertices: vertices, Edges: edges}, nil
}

// normalizeDirection accepts the entitygraph contract values
// ("outbound"/"inbound"/"any") case-insensitively and returns the AQL keyword
// to inline in the traversal query. An empty input defaults to OUTBOUND.
// Any other value is rejected before substitution to keep the AQL safe.
func normalizeDirection(d string) (string, error) {
	switch strings.ToUpper(d) {
	case "":
		return "OUTBOUND", nil
	case "OUTBOUND":
		return "OUTBOUND", nil
	case "INBOUND":
		return "INBOUND", nil
	case "ANY":
		return "ANY", nil
	default:
		return "", fmt.Errorf("TraverseGraph: invalid direction %q (want outbound|inbound|any)", d)
	}
}

// toRelationship converts a relationshipDoc and its ArangoDB _key to a
// [entitygraph.Relationship].
func toRelationship(doc relationshipDoc, key string) entitygraph.Relationship {
	r := entitygraph.Relationship{
		ID:         key,
		AgencyID:   doc.AgencyID,
		Name:       doc.Name,
		Properties: doc.Properties,
		CreatedAt:  doc.CreatedAt,
	}
	// Strip collection prefix from _from / _to to get plain entity IDs.
	r.FromID = stripCollectionPrefix(doc.From)
	r.ToID = stripCollectionPrefix(doc.To)
	if r.Properties == nil {
		r.Properties = make(map[string]any)
	}
	return r
}

// stripCollectionPrefix removes the "<collection>/" prefix from an ArangoDB
// document handle, returning only the document key.
func stripCollectionPrefix(handle string) string {
	for i := len(handle) - 1; i >= 0; i-- {
		if handle[i] == '/' {
			return handle[i+1:]
		}
	}
	return handle
}
