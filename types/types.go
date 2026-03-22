// Package types defines the shared domain types used across CodeVald
// microservices. These are pure data structures — no logic, no dependencies
// beyond the Go standard library. They represent concepts that cross service
// boundaries: route metadata, service registration payloads, and path-parameter
// bindings used by the CodeValdCross reverse proxy.
package types

import "time"

// PathBinding maps one URL path-parameter placeholder (as it appears in the
// route pattern, e.g. "agencyId") to the corresponding top-level field name in
// the gRPC request message (e.g. "agency_id"). CodeValdCross injects the
// runtime path value into the JSON request body before forwarding the call to
// the downstream service.
type PathBinding struct {
	// URLParam is the placeholder name from the URL pattern, e.g. "agencyId".
	URLParam string `json:"url_param"`
	// Field is the top-level proto field name in the request message, e.g. "agency_id".
	Field string `json:"field"`
}

// ConstantBinding injects a hardcoded field value into every gRPC request for
// a route, regardless of the HTTP request content. CodeValdCross writes this
// value into the JSON body before forwarding so that generic RPCs like
// ListEntities and CreateEntity receive the correct type_id or relationship
// name without the HTTP caller having to supply it.
type ConstantBinding struct {
	// Field is the proto field name to inject (e.g. "type_id", "name").
	Field string `json:"field"`
	// Value is the hardcoded value to set (e.g. "Goal", "belongs_to_agency").
	Value string `json:"value"`
}

// RouteInfo is the serialisable metadata for a single HTTP route declared by a
// downstream service at registration time. CodeValdCross stores these and uses
// GrpcMethod together with PathBindings and ConstantBindings when acting as a
// reverse proxy.
type RouteInfo struct {
	// Method is the HTTP verb (e.g. "GET", "POST").
	Method string `json:"method"`
	// Pattern is the URL pattern (e.g. "/{agencyId}/tasks/{taskId}/files").
	Pattern string `json:"pattern"`
	// Capability is the human-readable operation identifier the service declared
	// (e.g. "list_task_files"). Useful for introspection and logging.
	Capability string `json:"capability,omitempty"`
	// GrpcMethod is the fully-qualified gRPC method path CodeValdCross invokes
	// when this route is matched (e.g. "/codevaldwork.v1.TaskService/CreateTask").
	GrpcMethod string `json:"grpc_method,omitempty"`
	// PathBindings declares how URL path parameters map into the top-level
	// fields of the gRPC request message.
	PathBindings []PathBinding `json:"path_bindings,omitempty"`
	// ConstantBindings injects fixed field values into the gRPC request body at
	// dispatch time. Used to carry type_id, relationship name, and similar
	// values that are known at route-declaration time.
	ConstantBindings []ConstantBinding `json:"constant_bindings,omitempty"`
}

// ServiceRegistration is the Go domain representation of a downstream service's
// registration payload. It is used by CodeValdCross to track which services are
// alive, what pub/sub topics they produce and consume, and which HTTP routes
// they expose via the reverse proxy.
type ServiceRegistration struct {
	// ServiceName is the unique name of the registering service (e.g. "codevaldgit").
	ServiceName string
	// AgencyID is the agency this service instance is scoped to.
	// An empty string means the instance serves all agencies or is unscoped.
	AgencyID string
	// Addr is the gRPC address (host:port) at which the service is reachable.
	// CodeValdCross dials this address after registration.
	Addr string
	// Produces is the list of pub/sub topic identifiers this service publishes.
	Produces []string
	// Consumes is the list of pub/sub topic identifiers this service subscribes to.
	Consumes []string
	// Routes are the HTTP endpoints this service declared at registration time.
	// Always non-nil after a registration — empty slice when no routes were declared.
	Routes []RouteInfo `json:"routes"`
	// LastPing is the UTC timestamp of the most recent Register or Ping call
	// received from this service.
	LastPing time.Time
}
