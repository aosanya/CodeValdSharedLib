package types_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aosanya/CodeValdSharedLib/types"
)

func TestPathBinding_JSONRoundTrip(t *testing.T) {
	original := types.PathBinding{
		URLParam: "agencyId",
		Field:    "agency_id",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got types.PathBinding
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, original)
	}
}

func TestRouteInfo_JSONRoundTrip(t *testing.T) {
	original := types.RouteInfo{
		Method:     "GET",
		Pattern:    "/{agencyId}/tasks/{taskId}/files",
		Capability: "list_task_files",
		GrpcMethod: "/codevaldwork.v1.TaskService/ListTaskFiles",
		PathBindings: []types.PathBinding{
			{URLParam: "agencyId", Field: "agency_id"},
			{URLParam: "taskId", Field: "task_id"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got types.RouteInfo
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Method != original.Method || got.Pattern != original.Pattern ||
		got.Capability != original.Capability || got.GrpcMethod != original.GrpcMethod {
		t.Errorf("RouteInfo fields mismatch: got %+v, want %+v", got, original)
	}
	if len(got.PathBindings) != len(original.PathBindings) {
		t.Fatalf("PathBindings len: got %d, want %d", len(got.PathBindings), len(original.PathBindings))
	}
	for i, b := range original.PathBindings {
		if got.PathBindings[i] != b {
			t.Errorf("PathBindings[%d]: got %+v, want %+v", i, got.PathBindings[i], b)
		}
	}
}

func TestRouteInfo_OmitEmpty(t *testing.T) {
	r := types.RouteInfo{Method: "POST", Pattern: "/tasks"}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// capability, grpc_method, path_bindings must not appear when empty
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal map: %v", err)
	}
	for _, key := range []string{"capability", "grpc_method", "path_bindings"} {
		if _, ok := m[key]; ok {
			t.Errorf("expected key %q to be omitted when empty, but it was present", key)
		}
	}
}

func TestServiceRegistration_JSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	original := types.ServiceRegistration{
		ServiceName: "codevaldgit",
		AgencyID:    "agency-42",
		Addr:        "localhost:9001",
		Produces:    []string{"git.repo.created"},
		Consumes:    []string{"cross.task.requested"},
		Routes: []types.RouteInfo{
			{Method: "GET", Pattern: "/{agencyId}/repos"},
		},
		LastPing: now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got types.ServiceRegistration
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.ServiceName != original.ServiceName {
		t.Errorf("ServiceName: got %q, want %q", got.ServiceName, original.ServiceName)
	}
	if got.AgencyID != original.AgencyID {
		t.Errorf("AgencyID: got %q, want %q", got.AgencyID, original.AgencyID)
	}
	if got.Addr != original.Addr {
		t.Errorf("Addr: got %q, want %q", got.Addr, original.Addr)
	}
	if len(got.Produces) != 1 || got.Produces[0] != original.Produces[0] {
		t.Errorf("Produces: got %v, want %v", got.Produces, original.Produces)
	}
	if len(got.Consumes) != 1 || got.Consumes[0] != original.Consumes[0] {
		t.Errorf("Consumes: got %v, want %v", got.Consumes, original.Consumes)
	}
	if len(got.Routes) != 1 || got.Routes[0].Method != "GET" {
		t.Errorf("Routes: got %v, want %v", got.Routes, original.Routes)
	}
}

func TestServiceRegistration_EmptyRoutes(t *testing.T) {
	sr := types.ServiceRegistration{
		ServiceName: "codevaldwork",
		Routes:      []types.RouteInfo{},
	}
	data, err := json.Marshal(sr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got types.ServiceRegistration
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Routes == nil {
		t.Error("Routes should be non-nil after round-trip of empty slice")
	}
}
