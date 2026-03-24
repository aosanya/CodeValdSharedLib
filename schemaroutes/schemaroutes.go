// Package schemaroutes generates [types.RouteInfo] slices from a [types.Schema].
//
// Services that back entity storage with entitygraph call [RoutesFromSchema] at
// startup to obtain the full set of CRUD and relationship HTTP routes to
// register with CodeValdCross — eliminating the need for hand-maintained, per-
// type route declarations.
//
// # Route shape generated
//
// For each TypeDefinition with a non-empty PathSegment and non-empty EntityIDParam:
//
//	GET    {basePath}/{type.PathSegment}                                     → ListEntities
//	POST   {basePath}/{type.PathSegment}                                     → CreateEntity
//	GET    {basePath}/{type.PathSegment}/{type.EntityIDParam}                → GetEntity
//	PUT    {basePath}/{type.PathSegment}/{type.EntityIDParam}                → UpdateEntity   (mutable types only)
//	DELETE {basePath}/{type.PathSegment}/{type.EntityIDParam}                → DeleteEntity
//
// For each RelationshipDefinition with a non-empty PathSegment on a
// TypeDefinition that itself has a non-empty PathSegment and EntityIDParam:
//
//	GET    {basePath}/{type.PathSegment}/{type.EntityIDParam}/{rel.PathSegment}          → ListRelationships
//	POST   {basePath}/{type.PathSegment}/{type.EntityIDParam}/{rel.PathSegment}          → CreateRelationship
//	DELETE {basePath}/{type.PathSegment}/{type.EntityIDParam}/{rel.PathSegment}/{relId}  → DeleteRelationship
//
// TypeDefinitions with a non-empty PathSegment but an empty EntityIDParam only
// receive the collection-level routes (ListEntities, CreateEntity); per-entity
// and relationship routes are skipped.
package schemaroutes

import (
	"strings"
	"unicode"

	"github.com/aosanya/CodeValdSharedLib/types"
)

// RoutesFromSchema derives a complete set of HTTP [types.RouteInfo] entries
// from schema. It is intended to be called once at service startup and the
// result passed (together with any hand-written static routes) directly into
// the SharedLib registrar.
//
// Parameters:
//
//   - schema        — the service's active schema (e.g. DefaultAgencySchema())
//   - basePath      — path prefix up to and including the agency-ID placeholder,
//     e.g. "/agency/{agencyId}"
//   - agencyIDParam — the URL placeholder name for the agency ID inside
//     basePath, e.g. "agencyId"; automatically bound to the "agency_id"
//     field in every gRPC request
//   - grpcService   — fully-qualified gRPC service path whose generic CRUD
//     methods are called, e.g. "/codevaldagency.v1.EntityService"
func RoutesFromSchema(schema types.Schema, basePath, agencyIDParam, grpcService string) []types.RouteInfo {
	agencyBinding := types.PathBinding{URLParam: agencyIDParam, Field: "agency_id"}
	relBinding := types.PathBinding{URLParam: "relId", Field: "relationship_id"}

	var routes []types.RouteInfo

	for _, td := range schema.Types {
		if td.PathSegment == "" {
			continue
		}

		// Use the type-specific entity ID param; skip per-entity and relationship
		// routes entirely when EntityIDParam is not declared — the type only gets
		// collection-level routes (list, create).
		entityIDParam := td.EntityIDParam
		entityBinding := types.PathBinding{URLParam: entityIDParam, Field: "entity_id"}

		typePath := basePath + "/" + td.PathSegment
		typeName := toSnake(td.Name)

		typeConstant := []types.ConstantBinding{{Field: "type_id", Value: td.Name}}

		// Intermediate path params are {param} placeholders embedded in the
		// PathSegment itself (e.g. {draftId} in "drafts/{draftId}/goals").
		// They are NOT the entity ID param — they scope the collection to a
		// parent entity. Each is bound to "properties.<snake_case_param>" in the
		// gRPC request so that ListEntities can filter by that property.
		intermediatBindings := intermediatePathBindings(td.PathSegment, entityIDParam)

		// LIST all entities of this type.
		// Includes intermediate bindings so that Draft* types are scoped to the
		// correct draft (e.g. draftRefCode → properties.draft_ref_code).
		listBindings := append([]types.PathBinding{agencyBinding}, intermediatBindings...)
		routes = append(routes, types.RouteInfo{
			Method:           "GET",
			Pattern:          typePath,
			Capability:       "list_" + typeName,
			GrpcMethod:       grpcService + "/ListEntities",
			PathBindings:     listBindings,
			ConstantBindings: typeConstant,
		})

		// CREATE a new entity of this type.
		routes = append(routes, types.RouteInfo{
			Method:           "POST",
			Pattern:          typePath,
			Capability:       "create_" + typeName,
			GrpcMethod:       grpcService + "/CreateEntity",
			PathBindings:     []types.PathBinding{agencyBinding},
			ConstantBindings: typeConstant,
		})

		// Per-entity and relationship routes require EntityIDParam.
		if entityIDParam == "" {
			continue
		}

		entitySeg := "/{" + entityIDParam + "}"

		// GET a single entity by ID.
		routes = append(routes, types.RouteInfo{
			Method:           "GET",
			Pattern:          typePath + entitySeg,
			Capability:       "get_" + typeName,
			GrpcMethod:       grpcService + "/GetEntity",
			PathBindings:     []types.PathBinding{agencyBinding, entityBinding},
			ConstantBindings: typeConstant,
		})

		// UPDATE a single entity by ID — skipped for Immutable types
		// (UpdateEntity returns ErrImmutableType for those).
		if !td.Immutable {
			routes = append(routes, types.RouteInfo{
				Method:           "PUT",
				Pattern:          typePath + entitySeg,
				Capability:       "update_" + typeName,
				GrpcMethod:       grpcService + "/UpdateEntity",
				PathBindings:     []types.PathBinding{agencyBinding, entityBinding},
				ConstantBindings: typeConstant,
			})
		}

		// DELETE a single entity by ID.
		routes = append(routes, types.RouteInfo{
			Method:           "DELETE",
			Pattern:          typePath + entitySeg,
			Capability:       "delete_" + typeName,
			GrpcMethod:       grpcService + "/DeleteEntity",
			PathBindings:     []types.PathBinding{agencyBinding, entityBinding},
			ConstantBindings: typeConstant,
		})

		// Relationship routes for each declared edge with a PathSegment.
		for _, rel := range td.Relationships {
			if rel.PathSegment == "" {
				continue
			}

			relPath := typePath + entitySeg + "/" + rel.PathSegment
			relCap := typeName + "_" + rel.Name
			relNameConstant := []types.ConstantBinding{{Field: "name", Value: rel.Name}}

			// LIST all edges of this type from the source entity.
			routes = append(routes, types.RouteInfo{
				Method:           "GET",
				Pattern:          relPath,
				Capability:       "list_" + relCap,
				GrpcMethod:       grpcService + "/ListRelationships",
				PathBindings:     []types.PathBinding{agencyBinding, entityBinding},
				ConstantBindings: relNameConstant,
			})

			// CREATE a new edge from the source entity.
			routes = append(routes, types.RouteInfo{
				Method:           "POST",
				Pattern:          relPath,
				Capability:       "create_" + relCap,
				GrpcMethod:       grpcService + "/CreateRelationship",
				PathBindings:     []types.PathBinding{agencyBinding, entityBinding},
				ConstantBindings: relNameConstant,
			})

			// DELETE an edge by relationship ID — no constant bindings needed.
			routes = append(routes, types.RouteInfo{
				Method:       "DELETE",
				Pattern:      relPath + "/{relId}",
				Capability:   "delete_" + relCap,
				GrpcMethod:   grpcService + "/DeleteRelationship",
				PathBindings: []types.PathBinding{agencyBinding, entityBinding, relBinding},
			})
		}
	}

	return routes
}

// toSnake converts a PascalCase or camelCase string to snake_case.
//
//	"WorkItem"                → "work_item"
//	"AgencyPublicationStatus" → "agency_publication_status"
//	"Goal"                    → "goal"
func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// intermediatePathBindings extracts {param} placeholders from pathSegment that
// are not the entity ID param and returns PathBindings that map each URL param
// to "properties.<snake_case_param>" in the gRPC request.
//
// Example: pathSegment "drafts/{draftRefCode}/goals", entityIDParam "goalId"
// → [{URLParam: "draftRefCode", Field: "properties.draft_ref_code"}]
//
// This ensures that LIST requests for Draft* sub-types are automatically
// scoped by the draft_ref_code property, preventing cross-draft data leakage.
func intermediatePathBindings(pathSegment, entityIDParam string) []types.PathBinding {
	var bindings []types.PathBinding
	for _, seg := range strings.Split(pathSegment, "/") {
		if !strings.HasPrefix(seg, "{") || !strings.HasSuffix(seg, "}") {
			continue
		}
		param := seg[1 : len(seg)-1]
		if param == entityIDParam {
			continue
		}
		bindings = append(bindings, types.PathBinding{
			URLParam: param,
			Field:    "properties." + toSnake(param),
		})
	}
	return bindings
}
