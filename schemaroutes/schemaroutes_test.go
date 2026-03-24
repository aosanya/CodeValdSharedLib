package schemaroutes_test

import (
	"fmt"
	"testing"

	"github.com/aosanya/CodeValdSharedLib/schemaroutes"
	"github.com/aosanya/CodeValdSharedLib/types"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// findRoute returns the first RouteInfo in routes whose Method and Pattern both
// match, or nil if no match is found.
func findRoute(routes []types.RouteInfo, method, pattern string) *types.RouteInfo {
	for i := range routes {
		if routes[i].Method == method && routes[i].Pattern == pattern {
			return &routes[i]
		}
	}
	return nil
}

// hasConstantBinding reports whether route contains a ConstantBinding with the
// given field and value.
func hasConstantBinding(route types.RouteInfo, field, value string) bool {
	for _, cb := range route.ConstantBindings {
		if cb.Field == field && cb.Value == value {
			return true
		}
	}
	return false
}

// hasPathBinding reports whether route contains a PathBinding with the given
// URLParam and Field.
func hasPathBinding(route types.RouteInfo, urlParam, field string) bool {
	for _, pb := range route.PathBindings {
		if pb.URLParam == urlParam && pb.Field == field {
			return true
		}
	}
	return false
}

// countRoutes returns the number of routes whose Method and Pattern match the
// given pair.
func countRoutes(routes []types.RouteInfo, method, pattern string) int {
	n := 0
	for _, r := range routes {
		if r.Method == method && r.Pattern == pattern {
			n++
		}
	}
	return n
}

// ── empty / degenerate schema ─────────────────────────────────────────────────

func TestRoutesFromSchema_EmptySchema_ReturnsEmptySlice(t *testing.T) {
	routes := schemaroutes.RoutesFromSchema(
		types.Schema{ID: "empty", AgencyID: "agency-1"},
		"/agency/{agencyId}", "agencyId",
		"/codevaldagency.v1.EntityService",
	)
	if len(routes) != 0 {
		t.Errorf("got %d routes, want 0", len(routes))
	}
}

func TestRoutesFromSchema_TypeWithEmptyPathSegment_ProducesNoRoutes(t *testing.T) {
	schema := types.Schema{
		ID: "no-seg",
		Types: []types.TypeDefinition{
			{
				Name:        "Agency",
				PathSegment: "", // explicitly empty — no routes
				Properties: []types.PropertyDefinition{
					{Name: "name", Type: types.PropertyTypeString},
				},
			},
		},
	}
	routes := schemaroutes.RoutesFromSchema(schema, "/agency/{agencyId}", "agencyId",
		"/codevaldagency.v1.EntityService")
	if len(routes) != 0 {
		t.Errorf("got %d routes for type with empty PathSegment, want 0", len(routes))
	}
}

// ── collection-only type (PathSegment set, EntityIDParam empty) ──────────────

func TestRoutesFromSchema_TypeWithNoEntityIDParam_OnlyCollectionRoutes(t *testing.T) {
	schema := types.Schema{
		ID: "collection-only",
		Types: []types.TypeDefinition{
			{
				Name:        "Tag",
				PathSegment: "tags",
				// EntityIDParam intentionally empty → no per-entity or relationship routes
			},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	// Must have exactly 2 routes: GET /tags and POST /tags.
	if len(routes) != 2 {
		t.Fatalf("got %d routes, want 2 (list + create); routes: %v", len(routes), routePatterns(routes))
	}

	list := findRoute(routes, "GET", basePath+"/tags")
	if list == nil {
		t.Fatal("missing GET /tags")
	}
	create := findRoute(routes, "POST", basePath+"/tags")
	if create == nil {
		t.Fatal("missing POST /tags")
	}

	// Both must have type_id constant binding.
	for _, r := range routes {
		if !hasConstantBinding(*findRouteByCapability(routes, r.Capability), "type_id", "Tag") {
			t.Errorf("route %s %s missing type_id=Tag constant binding", r.Method, r.Pattern)
		}
	}
}

// ── full CRUD type (PathSegment + EntityIDParam set) ─────────────────────────

func TestRoutesFromSchema_MutableType_ProducesFiveCRUDRoutes(t *testing.T) {
	schema := types.Schema{
		ID: "full-crud",
		Types: []types.TypeDefinition{
			{
				Name:          "Goal",
				PathSegment:   "goals",
				EntityIDParam: "goalId",
			},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	// Expect: GET /goals, POST /goals, GET /goals/{goalId}, PUT /goals/{goalId}, DELETE /goals/{goalId}
	expected := []struct{ method, pattern string }{
		{"GET", basePath + "/goals"},
		{"POST", basePath + "/goals"},
		{"GET", basePath + "/goals/{goalId}"},
		{"PUT", basePath + "/goals/{goalId}"},
		{"DELETE", basePath + "/goals/{goalId}"},
	}
	for _, e := range expected {
		if r := findRoute(routes, e.method, e.pattern); r == nil {
			t.Errorf("missing route %s %s", e.method, e.pattern)
		}
	}
	if len(routes) != 5 {
		t.Errorf("got %d routes for mutable type, want 5; patterns: %v", len(routes), routePatterns(routes))
	}
}

func TestRoutesFromSchema_ImmutableType_NoPUTRoute(t *testing.T) {
	schema := types.Schema{
		ID: "immutable",
		Types: []types.TypeDefinition{
			{
				Name:          "Snapshot",
				PathSegment:   "snapshots",
				EntityIDParam: "snapshotId",
				Immutable:     true,
			},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	// PUT must be absent for immutable types.
	if r := findRoute(routes, "PUT", basePath+"/snapshots/{snapshotId}"); r != nil {
		t.Error("immutable type must not produce a PUT route")
	}

	// But GET/POST/GET(id)/DELETE should still be present.
	expected := []struct{ method, pattern string }{
		{"GET", basePath + "/snapshots"},
		{"POST", basePath + "/snapshots"},
		{"GET", basePath + "/snapshots/{snapshotId}"},
		{"DELETE", basePath + "/snapshots/{snapshotId}"},
	}
	for _, e := range expected {
		if r := findRoute(routes, e.method, e.pattern); r == nil {
			t.Errorf("missing route %s %s for immutable type", e.method, e.pattern)
		}
	}
	if len(routes) != 4 {
		t.Errorf("got %d routes for immutable type, want 4; patterns: %v", len(routes), routePatterns(routes))
	}
}

// ── relationship routes ───────────────────────────────────────────────────────

func TestRoutesFromSchema_RelationshipWithPathSegment_ThreeSubRoutes(t *testing.T) {
	schema := types.Schema{
		ID: "rel-routes",
		Types: []types.TypeDefinition{
			{
				Name:          "Agency",
				PathSegment:   "agencies",
				EntityIDParam: "agencyEntityId",
				Relationships: []types.RelationshipDefinition{
					{Name: "has_goal", ToType: "Goal", ToMany: true, PathSegment: "goals"},
				},
			},
			{Name: "Goal", PathSegment: "goals", EntityIDParam: "goalId"},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	relBase := basePath + "/agencies/{agencyEntityId}/goals"
	expected := []struct{ method, pattern string }{
		{"GET", relBase},
		{"POST", relBase},
		{"DELETE", relBase + "/{relId}"},
	}
	for _, e := range expected {
		if r := findRoute(routes, e.method, e.pattern); r == nil {
			t.Errorf("missing relationship route %s %s", e.method, e.pattern)
		}
	}
}

func TestRoutesFromSchema_RelationshipWithEmptyPathSegment_NoSubRoutes(t *testing.T) {
	schema := types.Schema{
		ID: "rel-no-seg",
		Types: []types.TypeDefinition{
			{
				Name:          "Agency",
				PathSegment:   "agencies",
				EntityIDParam: "agencyEntityId",
				Relationships: []types.RelationshipDefinition{
					{Name: "has_goal", ToType: "Goal", ToMany: true, PathSegment: ""},
				},
			},
			{Name: "Goal"},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	// No relationship sub-routes; only the 5 CRUD routes for Agency.
	for _, r := range routes {
		if r.Pattern == basePath+"/agencies/{agencyEntityId}/goals" {
			t.Errorf("unexpected relationship route for empty PathSegment: %s %s", r.Method, r.Pattern)
		}
	}
}

// ── constant and path bindings ────────────────────────────────────────────────

func TestRoutesFromSchema_EntityCRUDRoutes_HaveTypeIDConstantBinding(t *testing.T) {
	schema := types.Schema{
		ID: "bindings",
		Types: []types.TypeDefinition{
			{Name: "Workflow", PathSegment: "workflows", EntityIDParam: "workflowId"},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	for _, r := range routes {
		if !hasConstantBinding(r, "type_id", "Workflow") {
			t.Errorf("route %s %s missing constant binding type_id=Workflow", r.Method, r.Pattern)
		}
	}
}

func TestRoutesFromSchema_CollectionRoute_HasAgencyPathBinding(t *testing.T) {
	schema := types.Schema{
		ID: "agency-binding",
		Types: []types.TypeDefinition{
			{Name: "Goal", PathSegment: "goals", EntityIDParam: "goalId"},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	listRoute := findRoute(routes, "GET", basePath+"/goals")
	if listRoute == nil {
		t.Fatal("missing GET /goals")
	}
	if !hasPathBinding(*listRoute, "agencyId", "agency_id") {
		t.Error("GET /goals missing path binding agencyId→agency_id")
	}
}

func TestRoutesFromSchema_PerEntityRoute_HasEntityIDPathBinding(t *testing.T) {
	schema := types.Schema{
		ID: "entity-binding",
		Types: []types.TypeDefinition{
			{Name: "Goal", PathSegment: "goals", EntityIDParam: "goalId"},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	getRoute := findRoute(routes, "GET", basePath+"/goals/{goalId}")
	if getRoute == nil {
		t.Fatal("missing GET /goals/{goalId}")
	}
	if !hasPathBinding(*getRoute, "goalId", "entity_id") {
		t.Error("GET /goals/{goalId} missing path binding goalId→entity_id")
	}
}

func TestRoutesFromSchema_RelationshipRoutes_HaveNameConstantBinding(t *testing.T) {
	schema := types.Schema{
		ID: "rel-constant",
		Types: []types.TypeDefinition{
			{
				Name:          "Agency",
				PathSegment:   "agencies",
				EntityIDParam: "agencyEntityId",
				Relationships: []types.RelationshipDefinition{
					{Name: "has_goal", ToType: "Goal", ToMany: true, PathSegment: "goals"},
				},
			},
			{Name: "Goal"},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	listRel := findRoute(routes, "GET", basePath+"/agencies/{agencyEntityId}/goals")
	if listRel == nil {
		t.Fatal("missing GET relationship route")
	}
	if !hasConstantBinding(*listRel, "name", "has_goal") {
		t.Error("list relationship route missing constant binding name=has_goal")
	}
}

func TestRoutesFromSchema_RelationshipDeleteRoute_HasRelIdPathBinding(t *testing.T) {
	schema := types.Schema{
		ID: "rel-relid",
		Types: []types.TypeDefinition{
			{
				Name:          "Agency",
				PathSegment:   "agencies",
				EntityIDParam: "agencyEntityId",
				Relationships: []types.RelationshipDefinition{
					{Name: "has_goal", ToType: "Goal", ToMany: true, PathSegment: "goals"},
				},
			},
			{Name: "Goal"},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	deleteRel := findRoute(routes, "DELETE", basePath+"/agencies/{agencyEntityId}/goals/{relId}")
	if deleteRel == nil {
		t.Fatal("missing DELETE /agencies/{agencyEntityId}/goals/{relId}")
	}
	if !hasPathBinding(*deleteRel, "relId", "relationship_id") {
		t.Error("DELETE relationship route missing path binding relId→relationship_id")
	}
}

// ── intermediate path bindings (parent-scoped sub-types) ─────────────────────

func TestRoutesFromSchema_IntermediatePathBinding_MappedToProperty(t *testing.T) {
	// PathSegment "drafts/{draftId}/goals" has an intermediate {draftId} param
	// that scopes the entity to a parent draft. It must be bound to
	// "properties.draft_id" in the gRPC request.
	schema := types.Schema{
		ID: "intermediate",
		Types: []types.TypeDefinition{
			{
				Name:          "DraftGoal",
				PathSegment:   "drafts/{draftId}/goals",
				EntityIDParam: "goalId",
			},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	listRoute := findRoute(routes, "GET", basePath+"/drafts/{draftId}/goals")
	if listRoute == nil {
		t.Fatal("missing GET /drafts/{draftId}/goals")
	}
	if !hasPathBinding(*listRoute, "draftId", "properties.draft_id") {
		t.Errorf("list route missing intermediate binding draftId→properties.draft_id; bindings: %v",
			listRoute.PathBindings)
	}
}

func TestRoutesFromSchema_IntermediatePathBinding_NotInPerEntityRoute(t *testing.T) {
	// The per-entity route still carries the intermediate binding so that
	// CodeValdCross can inject draft_id into GetEntity/UpdateEntity requests.
	schema := types.Schema{
		ID: "intermediate-entity",
		Types: []types.TypeDefinition{
			{
				Name:          "DraftGoal",
				PathSegment:   "drafts/{draftId}/goals",
				EntityIDParam: "goalId",
			},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	getRoute := findRoute(routes, "GET", basePath+"/drafts/{draftId}/goals/{goalId}")
	if getRoute == nil {
		t.Fatal("missing GET /drafts/{draftId}/goals/{goalId}")
	}
	if !hasPathBinding(*getRoute, "goalId", "entity_id") {
		t.Error("per-entity route missing entity_id binding")
	}
}

// ── capability naming ─────────────────────────────────────────────────────────

func TestRoutesFromSchema_Capability_UsesSnakeCase(t *testing.T) {
	schema := types.Schema{
		ID: "capability",
		Types: []types.TypeDefinition{
			{Name: "WorkItem", PathSegment: "work-items", EntityIDParam: "workItemId"},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	listRoute := findRoute(routes, "GET", basePath+"/work-items")
	if listRoute == nil {
		t.Fatal("missing list route")
	}
	if listRoute.Capability != "list_work_item" {
		t.Errorf("Capability = %q, want %q", listRoute.Capability, "list_work_item")
	}
	createRoute := findRoute(routes, "POST", basePath+"/work-items")
	if createRoute == nil {
		t.Fatal("missing create route")
	}
	if createRoute.Capability != "create_work_item" {
		t.Errorf("Capability = %q, want %q", createRoute.Capability, "create_work_item")
	}
}

// ── gRPC method mapping ───────────────────────────────────────────────────────

func TestRoutesFromSchema_GRPCMethodMapping(t *testing.T) {
	schema := types.Schema{
		ID: "grpc-methods",
		Types: []types.TypeDefinition{
			{
				Name:          "Goal",
				PathSegment:   "goals",
				EntityIDParam: "goalId",
				Relationships: []types.RelationshipDefinition{
					{Name: "has_task", ToType: "Task", ToMany: true, PathSegment: "tasks"},
				},
			},
			{Name: "Task"},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/codevaldagency.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	cases := []struct {
		method, pattern, wantGRPC string
	}{
		{"GET", basePath + "/goals", svc + "/ListEntities"},
		{"POST", basePath + "/goals", svc + "/CreateEntity"},
		{"GET", basePath + "/goals/{goalId}", svc + "/GetEntity"},
		{"PUT", basePath + "/goals/{goalId}", svc + "/UpdateEntity"},
		{"DELETE", basePath + "/goals/{goalId}", svc + "/DeleteEntity"},
		{"GET", basePath + "/goals/{goalId}/tasks", svc + "/ListRelationships"},
		{"POST", basePath + "/goals/{goalId}/tasks", svc + "/CreateRelationship"},
		{"DELETE", basePath + "/goals/{goalId}/tasks/{relId}", svc + "/DeleteRelationship"},
	}

	for _, tc := range cases {
		r := findRoute(routes, tc.method, tc.pattern)
		if r == nil {
			t.Errorf("missing route %s %s", tc.method, tc.pattern)
			continue
		}
		if r.GrpcMethod != tc.wantGRPC {
			t.Errorf("%s %s: GrpcMethod = %q, want %q", tc.method, tc.pattern, r.GrpcMethod, tc.wantGRPC)
		}
	}
}

// ── multi-type schema ─────────────────────────────────────────────────────────

func TestRoutesFromSchema_MultipleTypes_AllGetRoutes(t *testing.T) {
	schema := types.Schema{
		ID: "multi-type",
		Types: []types.TypeDefinition{
			{Name: "Goal", PathSegment: "goals", EntityIDParam: "goalId"},
			{Name: "Workflow", PathSegment: "workflows", EntityIDParam: "workflowId"},
			{Name: "Internal", PathSegment: ""}, // no routes
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	// 5 routes per mutable type × 2 types + 0 for Internal = 10 routes.
	if len(routes) != 10 {
		t.Errorf("got %d routes, want 10; patterns: %v", len(routes), routePatterns(routes))
	}
}

func TestRoutesFromSchema_NoDuplicateRoutes(t *testing.T) {
	schema := types.Schema{
		ID: "no-dup",
		Types: []types.TypeDefinition{
			{Name: "Goal", PathSegment: "goals", EntityIDParam: "goalId"},
		},
	}
	basePath := "/agency/{agencyId}"
	svc := "/svc.v1.EntityService"
	routes := schemaroutes.RoutesFromSchema(schema, basePath, "agencyId", svc)

	seen := make(map[string]int)
	for _, r := range routes {
		key := fmt.Sprintf("%s %s", r.Method, r.Pattern)
		seen[key]++
	}
	for key, count := range seen {
		if count > 1 {
			t.Errorf("duplicate route %s appears %d times", key, count)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func routePatterns(routes []types.RouteInfo) []string {
	out := make([]string, len(routes))
	for i, r := range routes {
		out[i] = r.Method + " " + r.Pattern
	}
	return out
}

func findRouteByCapability(routes []types.RouteInfo, capability string) *types.RouteInfo {
	for i := range routes {
		if routes[i].Capability == capability {
			return &routes[i]
		}
	}
	return nil
}
