package entitygraph_test

import (
"errors"
"testing"

"github.com/aosanya/CodeValdSharedLib/entitygraph"
"github.com/aosanya/CodeValdSharedLib/types"
)

// testSchema builds a minimal Schema with Agency, Goal, and Workflow types for
// use in lookup and validation tests.
func testSchema() types.Schema {
return types.Schema{
ID:      "test-schema-v1",
Version: 1,
Tag:     "v1",
Types: []types.TypeDefinition{
{
Name:        "Agency",
DisplayName: "Agency",
PathSegment: "agencies",
Properties: []types.PropertyDefinition{
{Name: "name", Type: types.PropertyTypeString, Required: true},
},
Relationships: []types.RelationshipDefinition{
{Name: "has_goal", Label: "Goals", ToType: "Goal", ToMany: true, PathSegment: "goals"},
{Name: "has_workflow", Label: "Workflows", ToType: "Workflow", ToMany: true, PathSegment: "workflows"},
},
},
{
Name:        "Goal",
DisplayName: "Goal",
PathSegment: "goals",
Properties: []types.PropertyDefinition{
{Name: "title", Type: types.PropertyTypeString, Required: true},
},
Relationships: []types.RelationshipDefinition{
{Name: "belongs_to_agency", Label: "Agency", ToType: "Agency", ToMany: false},
},
},
{
Name:        "Workflow",
DisplayName: "Workflow",
PathSegment: "workflows",
Properties: []types.PropertyDefinition{
{Name: "name", Type: types.PropertyTypeString, Required: true},
},
Relationships: nil,
},
},
}
}

// ── FindTypeDef ──────────────────────────────────────────────────────────────

func TestFindTypeDef_ExistingType_ReturnsDef(t *testing.T) {
schema := testSchema()
td, err := entitygraph.FindTypeDef(schema, "Agency")
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if td.Name != "Agency" {
t.Errorf("got name %q, want %q", td.Name, "Agency")
}
}

func TestFindTypeDef_UnknownType_ReturnsError(t *testing.T) {
schema := testSchema()
_, err := entitygraph.FindTypeDef(schema, "NonExistent")
if err == nil {
t.Fatal("expected error, got nil")
}
}

func TestFindTypeDef_EmptySchema_ReturnsError(t *testing.T) {
_, err := entitygraph.FindTypeDef(types.Schema{ID: "empty"}, "Agency")
if err == nil {
t.Fatal("expected error, got nil")
}
}

// ── FindRelationshipDef ──────────────────────────────────────────────────────

func TestFindRelationshipDef_ExistingLabel_ReturnsDef(t *testing.T) {
schema := testSchema()
td, _ := entitygraph.FindTypeDef(schema, "Agency")
rd, err := entitygraph.FindRelationshipDef(td, "has_goal")
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if rd.ToType != "Goal" {
t.Errorf("got ToType %q, want %q", rd.ToType, "Goal")
}
if !rd.ToMany {
t.Error("expected ToMany = true for has_goal")
}
}

func TestFindRelationshipDef_FunctionalLabel_ToManyFalse(t *testing.T) {
schema := testSchema()
td, _ := entitygraph.FindTypeDef(schema, "Goal")
rd, err := entitygraph.FindRelationshipDef(td, "belongs_to_agency")
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if rd.ToMany {
t.Error("expected ToMany = false for belongs_to_agency")
}
}

func TestFindRelationshipDef_UnknownLabel_ReturnsError(t *testing.T) {
schema := testSchema()
td, _ := entitygraph.FindTypeDef(schema, "Agency")
_, err := entitygraph.FindRelationshipDef(td, "nonexistent_label")
if err == nil {
t.Fatal("expected error, got nil")
}
}

func TestFindRelationshipDef_TypeWithNoRelationships_ReturnsError(t *testing.T) {
td := types.TypeDefinition{Name: "Bare", Relationships: nil}
_, err := entitygraph.FindRelationshipDef(td, "has_goal")
if err == nil {
t.Fatal("expected error for type with no relationships, got nil")
}
}

// ── RelationshipDefinition field values ──────────────────────────────────────

func TestRelationshipDefinition_AllFields(t *testing.T) {
rd := types.RelationshipDefinition{
Name:        "has_snapshot",
Label:       "Snapshots",
ToType:      "AgencySnapshot",
ToMany:      true,
Required:    false,
Inverse:     "belongs_to_agency",
PathSegment: "snapshots",
}
if rd.Name != "has_snapshot" {
t.Errorf("Name: got %q", rd.Name)
}
if rd.ToType != "AgencySnapshot" {
t.Errorf("ToType: got %q", rd.ToType)
}
if rd.Inverse != "belongs_to_agency" {
t.Errorf("Inverse: got %q", rd.Inverse)
}
if rd.PathSegment != "snapshots" {
t.Errorf("PathSegment: got %q, want %q", rd.PathSegment, "snapshots")
}
}

// ── ValidateCreateRelationship ───────────────────────────────────────────────

func TestValidateCreateRelationship_ValidEdge(t *testing.T) {
schema := testSchema()
td, _ := entitygraph.FindTypeDef(schema, "Agency")
if err := entitygraph.ValidateCreateRelationship(td, "has_goal", "Goal"); err != nil {
t.Errorf("unexpected error for valid edge: %v", err)
}
}

func TestValidateCreateRelationship_UnknownLabel_ReturnsErrInvalidRelationship(t *testing.T) {
schema := testSchema()
td, _ := entitygraph.FindTypeDef(schema, "Agency")
err := entitygraph.ValidateCreateRelationship(td, "unknown_label", "Goal")
if !errors.Is(err, entitygraph.ErrInvalidRelationship) {
t.Errorf("got %v, want ErrInvalidRelationship", err)
}
}

func TestValidateCreateRelationship_WrongToType_ReturnsErrInvalidRelationship(t *testing.T) {
schema := testSchema()
td, _ := entitygraph.FindTypeDef(schema, "Agency")
err := entitygraph.ValidateCreateRelationship(td, "has_goal", "WrongType")
if !errors.Is(err, entitygraph.ErrInvalidRelationship) {
t.Errorf("got %v, want ErrInvalidRelationship", err)
}
}

func TestValidateCreateRelationship_FunctionalEdge(t *testing.T) {
schema := testSchema()
td, _ := entitygraph.FindTypeDef(schema, "Goal")
if err := entitygraph.ValidateCreateRelationship(td, "belongs_to_agency", "Agency"); err != nil {
t.Errorf("unexpected error for valid functional edge: %v", err)
}
}

// ── ValidateSchema ───────────────────────────────────────────────────────────

func TestValidateSchema_ValidSchema_NoError(t *testing.T) {
if err := entitygraph.ValidateSchema(testSchema()); err != nil {
t.Errorf("unexpected error for valid schema: %v", err)
}
}

func TestValidateSchema_EmptySchema_NoError(t *testing.T) {
s := types.Schema{ID: "empty", AgencyID: "agency-1"}
if err := entitygraph.ValidateSchema(s); err != nil {
t.Errorf("unexpected error for empty schema: %v", err)
}
}

func TestValidateSchema_DuplicateTypeName_ReturnsError(t *testing.T) {
s := types.Schema{
ID:       "dup-names",
AgencyID: "agency-1",
Types: []types.TypeDefinition{
{Name: "Pump"},
{Name: "Pump"},
},
}
if err := entitygraph.ValidateSchema(s); err == nil {
t.Fatal("expected error for duplicate type name, got nil")
}
}

func TestValidateSchema_DuplicateTypePathSegment_ReturnsError(t *testing.T) {
s := types.Schema{
ID:       "dup-path-segs",
AgencyID: "agency-1",
Types: []types.TypeDefinition{
{Name: "Pump", PathSegment: "devices"},
{Name: "Sensor", PathSegment: "devices"},
},
}
if err := entitygraph.ValidateSchema(s); err == nil {
t.Fatal("expected error for duplicate type PathSegment, got nil")
}
}

func TestValidateSchema_EmptyPathSegment_AllowedMultipleTypes(t *testing.T) {
s := types.Schema{
ID:       "empty-path-segs",
AgencyID: "agency-1",
Types: []types.TypeDefinition{
{Name: "Pump", PathSegment: ""},
{Name: "Sensor", PathSegment: ""},
},
}
if err := entitygraph.ValidateSchema(s); err != nil {
t.Errorf("multiple types with empty PathSegment should be allowed: %v", err)
}
}

func TestValidateSchema_InverseToTypeNotFound_ReturnsError(t *testing.T) {
s := types.Schema{
ID:       "bad-inverse-totype",
AgencyID: "agency-1",
Types: []types.TypeDefinition{
{
Name: "Agency",
Relationships: []types.RelationshipDefinition{
{Name: "has_goal", ToType: "Goal", ToMany: true, Inverse: "belongs_to_agency"},
},
},
// Goal type is missing — ToType not found in schema
},
}
if err := entitygraph.ValidateSchema(s); err == nil {
t.Fatal("expected error when ToType not found in schema, got nil")
}
}

func TestValidateSchema_InverseNotDeclaredOnToType_ReturnsError(t *testing.T) {
s := types.Schema{
ID:       "missing-inverse-decl",
AgencyID: "agency-1",
Types: []types.TypeDefinition{
{
Name: "Agency",
Relationships: []types.RelationshipDefinition{
{Name: "has_goal", ToType: "Goal", ToMany: true, Inverse: "belongs_to_agency"},
},
},
{
Name:          "Goal",
Relationships: nil, // does NOT declare belongs_to_agency
},
},
}
if err := entitygraph.ValidateSchema(s); err == nil {
t.Fatal("expected error when inverse not declared on ToType, got nil")
}
}

func TestValidateSchema_ValidInverse_NoError(t *testing.T) {
s := types.Schema{
ID:       "valid-inverse",
AgencyID: "agency-1",
Types: []types.TypeDefinition{
{
Name: "Agency",
Relationships: []types.RelationshipDefinition{
{Name: "has_goal", ToType: "Goal", ToMany: true, Inverse: "belongs_to_agency"},
},
},
{
Name: "Goal",
Relationships: []types.RelationshipDefinition{
{Name: "belongs_to_agency", ToType: "Agency", ToMany: false},
},
},
},
}
if err := entitygraph.ValidateSchema(s); err != nil {
t.Errorf("unexpected error for valid inverse: %v", err)
}
}

func TestValidateSchema_DuplicateRelationshipPathSegment_ReturnsError(t *testing.T) {
s := types.Schema{
ID:       "dup-rel-path-segs",
AgencyID: "agency-1",
Types: []types.TypeDefinition{
{
Name: "Agency",
Relationships: []types.RelationshipDefinition{
{Name: "has_goal", ToType: "Goal", ToMany: true, PathSegment: "items"},
{Name: "has_workflow", ToType: "Workflow", ToMany: true, PathSegment: "items"},
},
},
{Name: "Goal"},
{Name: "Workflow"},
},
}
if err := entitygraph.ValidateSchema(s); err == nil {
t.Fatal("expected error for duplicate relationship PathSegment, got nil")
}
}

func TestValidateSchema_EmptyRelPathSegment_AllowedMultiple(t *testing.T) {
s := types.Schema{
ID:       "empty-rel-path-segs",
AgencyID: "agency-1",
Types: []types.TypeDefinition{
{
Name: "Agency",
Relationships: []types.RelationshipDefinition{
{Name: "has_goal", ToType: "Goal", ToMany: true, PathSegment: ""},
{Name: "has_workflow", ToType: "Workflow", ToMany: true, PathSegment: ""},
},
},
{Name: "Goal"},
{Name: "Workflow"},
},
}
if err := entitygraph.ValidateSchema(s); err != nil {
t.Errorf("multiple relationships with empty PathSegment should be allowed: %v", err)
}
}
